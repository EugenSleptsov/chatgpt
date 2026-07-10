package service

import (
	"GPTBot/domain/ai"
	chatdomain "GPTBot/domain/chat"
	"fmt"
	"log"
	"strings"
)

const fallbackResponse = "Произошла ошибка с получением ответа, пожалуйста, попробуйте позднее"

type CostFunc func(tierID string, inputTokens, outputTokens int) float64

// extractUsage builds a usage step from an API response: token counts, cost
// (via costFn), plus the phase label and the tools that were sent.
func extractUsage(resp *ai.Response, tierID, phase string, costFn CostFunc, toolNames ...string) UsageStep {
	step := UsageStep{Phase: phase, ToolNames: toolNames}
	if resp == nil {
		return step
	}
	if costFn != nil {
		step.Cost = costFn(tierID, resp.Usage.InputTokens, resp.Usage.OutputTokens)
	}
	step.InputTokens = resp.Usage.InputTokens
	step.OutputTokens = resp.Usage.OutputTokens
	step.TotalTokens = resp.Usage.TotalTokens
	if resp.Usage.InputTokensDetails != nil {
		step.CachedTokens = resp.Usage.InputTokensDetails.CachedTokens
	}
	if resp.Usage.OutputTokensDetails != nil {
		step.ReasoningTokens = resp.Usage.OutputTokensDetails.ReasoningTokens
	}
	return step
}

// GPTService is the compatibility facade used by commands and executors.
// The concrete work is split into smaller helpers: Complete owns the stateful
// chat flow, OneShotService owns stateless AI calls, and ToolRunner owns the
// tool loop.
type GPTService struct {
	GptClient ai.Client
	Compact   *CompactService    // auto-compact (may be nil)
	CostFn    CostFunc           // provider-specific token cost calculator
	ImageCost float64            // provider-specific per-image generation cost (USD)
	Progress  ProgressReporter   // status/verbose messages (may be nil)
	Archive   chatdomain.Archive // supermemory raw transcript store (may be nil)
}

func NewGPTService(client ai.Client, compact *CompactService, costFn CostFunc, imageCost float64, progress ProgressReporter, archive chatdomain.Archive) *GPTService {
	return &GPTService{
		GptClient: client,
		Compact:   compact,
		CostFn:    costFn,
		ImageCost: imageCost,
		Progress:  progress,
		Archive:   archive,
	}
}

func (s *GPTService) oneShot() OneShotService {
	return OneShotService{
		Client:    s.GptClient,
		CostFn:    s.CostFn,
		ImageCost: s.ImageCost,
	}
}

func (s *GPTService) toolRunner() ToolRunner {
	return ToolRunner{
		Client:    s.GptClient,
		CostFn:    s.CostFn,
		ImageCost: s.ImageCost,
		Progress:  s.Progress,
		Archive:   s.Archive,
	}
}

type ChatResult struct {
	Text      string
	Images    []ImageResult
	Audio     []byte
	AudioText string
	Usage     TokenUsage
}

type ImageResult struct {
	Data []byte
}

func buildHistoryContent(r *ChatResult) string {
	var parts []string
	if r.Text != "" {
		parts = append(parts, r.Text)
	}
	for range r.Images {
		parts = append(parts, "[Сгенерирована картинка]")
	}
	if r.AudioText != "" {
		parts = append(parts, fmt.Sprintf("[Сгенерировано аудио: «%s»]", r.AudioText))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n")
}

// memorySections combines the supermemory index and advisor sections into the
// single prompt string threaded through instructions, compaction and token
// metrics.
func memorySections(chat *chatdomain.Chat) string {
	return JoinPrompts(SupermemoryPrompt(chat), AdvisorPrompt(chat))
}

func (s *GPTService) buildInstructions(session *chatdomain.Session, chat *chatdomain.Chat) string {
	return BuildInstructions(session, memorySections(chat), &PromptContext{
		ChatTitle:   chat.Title,
		IsGroup:     chat.ChatID < 0, // Telegram convention: group IDs are negative
		UseMarkdown: chat.Settings.UseMarkdown,
		Location:    chat.Location(),
		Supermemory: chat.Settings.Supermemory,
	})
}

// failSession records a fallback response in the session history and returns
// a ChatResult with the given text. Used by Complete on error paths.
func (s *GPTService) failSession(session *chatdomain.Session, text string) *ChatResult {
	AttachResponse(session, chatdomain.Message{Role: "assistant", Content: text})
	return &ChatResult{Text: text}
}

// costLimitResponse is returned when the chat has exceeded its daily spending cap.
const costLimitResponse = "⚠️ Дневной лимит расходов для этого чата исчерпан. Попробуйте завтра или попросите администратора увеличить лимит."

// Complete runs the GPT pipeline on the active session: calls GPT with the
// current history, handles tool calls, records metrics and attaches the
// assistant response. The caller is responsible for preparing the session
// before calling Complete.
func (s *GPTService) Complete(chat *chatdomain.Chat) (*ChatResult, error) {
	if chat.CostLimitExceeded(chat.Settings.CostLimitUSD) {
		session := chat.ActiveSession()
		return s.failSession(session, costLimitResponse), nil
	}

	session := chat.ActiveSession()

	// Supermemory: archive new history lines before compaction so evicted
	// entries carry their raw-source ranges. No-op when the feature is off.
	ArchiveHistory(s.Archive, chat, session)

	if s.Compact != nil {
		memPrompt := memorySections(chat)
		if s.Compact.ShouldCompact(session, memPrompt, session.LastInputTokens) {
			compactUsage, compactErr := s.Compact.Compact(chat, session, memPrompt)
			if compactErr != nil {
				log.Printf("[Complete] auto-compact failed: %v (proceeding without compaction)", compactErr)
			} else if compactUsage != nil {
				chat.AccumulateCost(compactUsage.Cost, compactUsage.InputTokens, compactUsage.OutputTokens)
				session.LastInputTokens = 0
			}
		}
		// Supermemory: fold the index into parent nodes when it outgrows the
		// prompt budget (no-op unless enabled and over the limit).
		if metaUsage, metaErr := s.Compact.MetaCompact(chat, session.Model); metaErr != nil {
			log.Printf("[Complete] meta-compact failed: %v (proceeding)", metaErr)
		} else if metaUsage != nil {
			chat.AccumulateCost(metaUsage.Cost, metaUsage.InputTokens, metaUsage.OutputTokens)
		}
	}

	messages := HistoryMessages(session)
	instructions := s.buildInstructions(session, chat)
	tools := toolsForChat(chat)

	payload, err := s.GptClient.CallGPT(messages, session.Model, instructions, tools...)
	if err != nil {
		log.Printf("[Complete] GPT error: %v", err)
		return s.failSession(session, fallbackResponse), err
	}

	result, err := s.toolRunner().Run(payload, session.Model, instructions, chat, tools, "GPT")
	if result == nil {
		result = &ChatResult{Text: fallbackResponse}
	}
	result.Usage.Input = computeInputMetrics(session, memorySections(chat), tools)

	chat.AccumulateCost(result.Usage.Cost, result.Usage.InputTokens, result.Usage.OutputTokens)

	// Save the last call's input_tokens as the current context size for the
	// next auto-compact threshold check. Summed input tokens would overcount
	// tool-loop continuations and compact too early.
	session.LastInputTokens = result.Usage.lastCallInputTokens

	AttachResponse(session, chatdomain.Message{Role: "assistant", Content: buildHistoryContent(result)})
	// Archive the just-attached response line (extends the entry's range).
	ArchiveHistory(s.Archive, chat, session)
	return result, err
}

// announceTools is kept as a small compatibility wrapper for package tests.
func (s *GPTService) announceTools(chat *chatdomain.Chat, response *ai.Response, calls []ai.ToolCall) {
	s.toolRunner().announceTools(chat, response, calls)
}
