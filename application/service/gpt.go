package service

import (
	"GPTBot/domain/ai"
	chatdomain "GPTBot/domain/chat"
	"fmt"
	"log"
	"strings"
)

const fallbackResponse = "Произошла ошибка с получением ответа, пожалуйста, попробуйте позднее"

func extractUsage(resp *ai.Response, tierID string, costFn func(string, int, int) float64) RawUsage {
	if resp == nil {
		return RawUsage{}
	}
	var cost float64
	if costFn != nil {
		cost = costFn(tierID, resp.Usage.InputTokens, resp.Usage.OutputTokens)
	}
	raw := RawUsage{
		InputTokens:  resp.Usage.InputTokens,
		OutputTokens: resp.Usage.OutputTokens,
		TotalTokens:  resp.Usage.TotalTokens,
		Cost:         cost,
	}
	if resp.Usage.InputTokensDetails != nil {
		raw.CachedTokens = resp.Usage.InputTokensDetails.CachedTokens
	}
	if resp.Usage.OutputTokensDetails != nil {
		raw.ReasoningTokens = resp.Usage.OutputTokensDetails.ReasoningTokens
	}
	return raw
}

type GPTService struct {
	GptClient ai.Client
	History   *HistoryService
	Memory    *MemoryService
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
	return s.History.BuildInstructions(session, s.Memory.BuildPrompt(chat))
}

// failSession records a fallback response in the session history and returns
// a ChatResult with the given text. Used by completeSession on error paths.
func (s *GPTService) failSession(session *chatdomain.Session, text string) *ChatResult {
	s.History.AttachResponse(session, chatdomain.Message{Role: "assistant", Content: text})
	return &ChatResult{Text: text}
}

// Complete runs the GPT pipeline on the active session: calls GPT with the
// current history, handles tool calls, records metrics and attaches the
// assistant response. The caller is responsible for preparing the session
// (appending user messages, checking history, etc.) before calling Complete.
func (s *GPTService) Complete(chat *chatdomain.Chat) (*ChatResult, error) {
	session := chat.ActiveSession()
	messages := s.History.Messages(session)
	instructions := s.buildInstructions(session, chat)

	payload, err := s.GptClient.CallGPT(messages, session.Model, instructions, chatToolsLight...)
	if err != nil {
		log.Printf("[Complete] GPT error: %v", err)
		return s.failSession(session, fallbackResponse), err
	}

	payload, tools, preflight, err := s.probeAndUpgrade(payload, session.Model, instructions, chat, "Complete")
	if err != nil {
		result := s.failSession(session, fallbackResponse)
		result.Usage = preflight
		return result, err
	}

	initialPhase := "GPT"
	if len(preflight.Steps) > 0 {
		initialPhase = preflight.upgradePhase
	}
	result, err := s.toolLoop(payload, session.Model, instructions, chat, tools, initialPhase)
	if result == nil {
		result = &ChatResult{Text: fallbackResponse}
	}
	result.Usage.prepend(preflight)
	result.Usage.Input = computeInputMetrics(session, s.Memory.BuildPrompt(chat), chatToolsLight)

	s.History.AttachResponse(session, chatdomain.Message{Role: "assistant", Content: buildHistoryContent(result)})
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
	result.Usage.accumulate(extractUsage(response, model, s.CostFn), initialPhase, tNames...)
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
		result.Usage.accumulate(extractUsage(response, model, s.CostFn), fmt.Sprintf("Continue (%s)", strings.Join(calledNames, ", ")), tNames...)
	}
	log.Printf("[ToolLoop] max iterations (%d) reached", maxToolIterations)
	s.collectImages(response, result)
	result.Text = strings.TrimSpace(response.OutputText())
	if result.Text == "" {
		result.Text = fallbackResponse
	}
	return result, nil
}
