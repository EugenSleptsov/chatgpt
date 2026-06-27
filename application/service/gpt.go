package service

import (
	"GPTBot/domain/ai"
	chatdomain "GPTBot/domain/chat"
	"fmt"
	"log"
	"strings"
)

const fallbackResponse = "Произошла ошибка с получением ответа, пожалуйста, попробуйте позднее"

// extractUsage builds a usage step from an API response: token counts, cost
// (via costFn), plus the phase label and the tools that were sent.
func extractUsage(resp *ai.Response, tierID, phase string, costFn func(string, int, int) float64, toolNames ...string) UsageStep {
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

type GPTService struct {
	GptClient ai.Client
	Compact   *CompactService                                            // auto-compact (may be nil)
	CostFn    func(tierID string, inputTokens, outputTokens int) float64 // provider-specific token cost calculator
	ImageCost float64                                                    // provider-specific per-image generation cost (USD)
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
func (s *GPTService) buildInstructions(session *chatdomain.Session, chat *chatdomain.Chat) string {
	return BuildInstructions(session, MemoryPrompt(chat), &PromptContext{
		ChatTitle:   chat.Title,
		IsGroup:     chat.ChatID < 0, // Telegram convention: group IDs are negative
		UseMarkdown: chat.Settings.UseMarkdown,
	})
}

// failSession records a fallback response in the session history and returns
// a ChatResult with the given text. Used by completeSession on error paths.
func (s *GPTService) failSession(session *chatdomain.Session, text string) *ChatResult {
	AttachResponse(session, chatdomain.Message{Role: "assistant", Content: text})
	return &ChatResult{Text: text}
}

// costLimitResponse is returned when the chat has exceeded its daily spending cap.
const costLimitResponse = "⚠️ Дневной лимит расходов для этого чата исчерпан. Попробуйте завтра или попросите администратора увеличить лимит."

// Complete runs the GPT pipeline on the active session: calls GPT with the
// current history, handles tool calls, records metrics and attaches the
// assistant response. The caller is responsible for preparing the session
// (appending user messages, checking history, etc.) before calling Complete.
//
// checks the cumulative daily spend before calling the API.
func (s *GPTService) Complete(chat *chatdomain.Chat) (*ChatResult, error) {
	// Cost guard: refuse to call API if daily limit exceeded.
	if chat.CostLimitExceeded(chat.Settings.CostLimitUSD) {
		session := chat.ActiveSession()
		return s.failSession(session, costLimitResponse), nil
	}

	session := chat.ActiveSession()

	// Auto-compact: if context is approaching the model's limit, summarize
	// old messages before sending. Uses real API token count from last
	// response when available (like Claude Code's tokenCountWithEstimation).
	if s.Compact != nil {
		memPrompt := MemoryPrompt(chat)
		if s.Compact.ShouldCompact(session, memPrompt, session.LastInputTokens) {
			compactUsage, compactErr := s.Compact.Compact(session, memPrompt)
			if compactErr != nil {
				log.Printf("[Complete] auto-compact failed: %v (proceeding without compaction)", compactErr)
			} else if compactUsage != nil {
				chat.AccumulateCost(compactUsage.Cost, compactUsage.InputTokens, compactUsage.OutputTokens)
				session.LastInputTokens = 0 // reset after compaction
			}
		}
	}

	messages := HistoryMessages(session)
	instructions := s.buildInstructions(session, chat)

	payload, err := s.GptClient.CallGPT(messages, session.Model, instructions, chatTools...)
	if err != nil {
		log.Printf("[Complete] GPT error: %v", err)
		return s.failSession(session, fallbackResponse), err
	}

	result, err := s.toolLoop(payload, session.Model, instructions, chat, chatTools, "GPT")
	if result == nil {
		result = &ChatResult{Text: fallbackResponse}
	}
	result.Usage.Input = computeInputMetrics(session, MemoryPrompt(chat), chatTools)

	// Accumulate cost on the chat (daily rolling counter).
	chat.AccumulateCost(result.Usage.Cost, result.Usage.InputTokens, result.Usage.OutputTokens)

	// Save real API input_tokens for next auto-compact threshold check.
	// Claude Code's tokenCountWithEstimation prefers the last API response's
	// usage.input_tokens over rough character-based estimates. Use the LAST
	// call's input tokens (current context size), not the summed total — the
	// sum inflates with each tool-loop iteration and would compact prematurely.
	session.LastInputTokens = result.Usage.lastCallInputTokens

	AttachResponse(session, chatdomain.Message{Role: "assistant", Content: buildHistoryContent(result)})
	return result, err
}

const maxToolIterations = 5

// collectImages extracts image data from the response and records their cost.
func (s *GPTService) collectImages(response *ai.Response, result *ChatResult) {
	for _, imgData := range response.ImageResults() {
		result.Images = append(result.Images, ImageResult{Data: imgData})
		result.Usage.addFixedCost("DALL-E (image)", s.ImageCost)
	}
}

func (s *GPTService) toolLoop(response *ai.Response, model, instructions string, chat *chatdomain.Chat, tools []ai.Tool, initialPhase string) (*ChatResult, error) {
	result := &ChatResult{}
	tNames := toolNamesFromTools(tools)
	result.Usage.add(extractUsage(response, model, initialPhase, s.CostFn, tNames...))
	for i := 0; i < maxToolIterations; i++ {
		s.collectImages(response, result)
		calls := response.ToolCalls()
		if len(calls) == 0 {
			result.Text = strings.TrimSpace(response.OutputText())
			return result, nil
		}
		if text := strings.TrimSpace(response.OutputText()); text != "" {
			result.Text = text
		}
		log.Printf("[ToolLoop] iteration %d: %d tool call(s)", i+1, len(calls))
		outputs := make([]ai.ToolCallOutput, 0, len(calls))
		for _, tc := range calls {
			output := s.executeSingleToolCall(tc, result, chat)
			outputs = append(outputs, ai.NewToolCallOutput(tc.ID, output))
		}
		var err error
		response, err = s.GptClient.ContinueWithToolOutputs(response.ID, outputs, model, instructions, tools...)
		if err != nil {
			log.Printf("[ToolLoop] error continuing response: %v", err)
			if result.Text == "" {
				result.Text = fallbackResponse
			}
			return result, err
		}
		calledNames := make([]string, 0, len(calls))
		for _, tc := range calls {
			calledNames = append(calledNames, tc.Name)
		}
		result.Usage.add(extractUsage(response, model, fmt.Sprintf("Continue (%s)", strings.Join(calledNames, ", ")), s.CostFn, tNames...))
	}
	log.Printf("[ToolLoop] max iterations (%d) reached", maxToolIterations)
	s.collectImages(response, result)
	result.Text = strings.TrimSpace(response.OutputText())
	if result.Text == "" {
		result.Text = fallbackResponse
	}
	return result, nil
}
