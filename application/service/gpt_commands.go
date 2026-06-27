package service

import (
	"GPTBot/domain/ai"
	chatdomain "GPTBot/domain/chat"
	"fmt"
	"strings"
)

// --- One-shot GPT operations ---
//
// The methods below are stateless: they do NOT touch conversation history
// (text commands, image generation/analysis, auto-reply decisions). They live
// on GPTService alongside the stateful chat pipeline (Complete) because both
// share the same client and pricing config.

// GPTCommand sends a one-shot system+user prompt pair to GPT and returns
// the response text with token usage. Does not touch chat history.
func (s *GPTService) GPTCommand(model string, systemPrompt, userPrompt string) (string, TokenUsage, error) {
	payload, err := s.GptClient.CallGPT([]ai.Message{
		{Role: "user", Content: []ai.Content{{Type: ai.TypeInputText, Text: userPrompt}}},
	}, model, systemPrompt)

	if err != nil {
		return "", TokenUsage{}, err
	}

	var usage TokenUsage
	usage.add(extractUsage(payload, model, "GPT", s.CostFn))

	if text := strings.TrimSpace(payload.OutputText()); text != "" {
		return text, usage, nil
	}
	return fallbackResponse, usage, nil
}

// GenerateImage creates an image from a prompt and returns the PNG bytes along
// with an AI-enhanced caption and accumulated usage/cost.
func (s *GPTService) GenerateImage(model string, prompt string) (imageData []byte, caption string, usage TokenUsage, err error) {
	imageData, err = s.GptClient.GenerateImage(prompt, ai.ImageSize1024)
	if err != nil {
		return nil, "", usage, err
	}
	usage.addFixedCost("gpt-image (image)", s.ImageCost)

	caption = prompt
	payload, err := s.GptClient.CallGPT([]ai.Message{
		{Role: "user", Content: fmt.Sprintf("Please improve this prompt: \"%s\". Answer with improved prompt only. Keep prompt at most 200 characters long. Your prompt must be in one sentence.", prompt)},
	}, model, "You are an assistant that generates natural language description (prompt) for an artificial intelligence (AI) that generates images")
	if err == nil {
		usage.add(extractUsage(payload, model, "GPT (caption)", s.CostFn))
		if text := strings.TrimSpace(payload.OutputText()); text != "" {
			caption = text
		}
	}

	return imageData, caption, usage, nil
}

// AnalyzeImage sends an image URL with a prompt to GPT Vision and returns the response.
func (s *GPTService) AnalyzeImage(imageURL, prompt string) (string, error) {
	messages := []ai.Message{
		{Role: "user", Content: []ai.Content{
			{Type: ai.TypeInputText, Text: prompt},
			{Type: ai.TypeInputImage, ImageUrl: imageURL},
		}},
	}

	payload, err := s.GptClient.CallGPT(messages, ai.VisionTierID, "")
	if err != nil {
		return "", err
	}

	if text := strings.TrimSpace(payload.OutputText()); text != "" {
		return text, nil
	}
	return fallbackResponse, nil
}

// DefaultAutoReplyPersona is the built-in persona used when neither the chat
// nor the config provides a custom one.
const DefaultAutoReplyPersona = `You are an active participant in a group chat. Your name is the bot mentioned in the conversation.`

// autoReplyDecisionTemplate is the fixed YES/NO instruction part
// that is always appended after the persona.
const autoReplyDecisionTemplate = `Look at the recent messages and decide if you should respond.
Reply YES if:
- someone is talking to you or about you (even without @mention — e.g. "бот, ответь")
- there is a factual question no one answered
- you can add something genuinely useful or funny
- someone shared news/link and no one commented
Reply NO only if:
- the conversation is purely between other people and doesn't need your input
- your last message already addressed the topic and nothing new was added
Respond with exactly one word: YES or NO.`

// buildAutoReplyPrompt combines the persona with the fixed decision template.
func buildAutoReplyPrompt(persona string) string {
	if persona == "" {
		persona = DefaultAutoReplyPersona
	}
	return persona + "\n\n" + autoReplyDecisionTemplate
}

// ShouldAutoReply asks GPT whether the bot should proactively join the group conversation.
// It looks at the last few history entries and asks for a YES/NO decision.
// The persona parameter describes the bot's role/personality; if empty, the built-in default is used.
func (s *GPTService) ShouldAutoReply(chat *chatdomain.Chat, persona string) (bool, string, error) {
	session := chat.ActiveSession()

	const lookback = 10
	history := session.History
	if len(history) > lookback {
		history = history[len(history)-lookback:]
	}

	messages := chatdomain.ToGPTMessages(history)
	if len(messages) == 0 {
		return false, "история пуста", nil
	}

	systemPrompt := buildAutoReplyPrompt(persona)
	payload, err := s.GptClient.CallGPT(messages, session.Model, systemPrompt)
	if err != nil {
		return false, "ошибка GPT", err
	}
	answer := strings.TrimSpace(payload.OutputText())
	yes := strings.HasPrefix(strings.ToUpper(answer), "YES")
	return yes, answer, nil
}
