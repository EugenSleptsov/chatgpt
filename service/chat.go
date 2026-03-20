// Package service contains transport-agnostic business logic.
// GPTService wraps GPT API calls (completions, images, logs)
// and knows nothing about Telegram or any other transport.
package service

import (
	"GPTBot/api/gpt"
	"GPTBot/api/logger"
	"GPTBot/storage"
	"GPTBot/util"
	"fmt"
	"strings"
)

const fallbackResponse = "Произошла ошибка с получением ответа, пожалуйста, попробуйте позднее"

// GPTService encapsulates GPT business logic (completions, image generation, logs),
// independent of any transport layer (Telegram, etc.).
type GPTService struct {
	GptClient gpt.Client
	Log       logger.Log
	ErrorLog  logger.ErrorLog
}

// ChatCompletion appends a user message to chat history, sends the
// conversation context to GPT, stores the assistant response in history
// and returns it. On GPT failure a fallback message is returned.
func (s *GPTService) ChatCompletion(chat *storage.Chat, userText string) string {
	session := chat.ActiveSession()

	entry := &storage.ConversationEntry{
		Prompt: storage.Message{Role: "user", Content: userText},
	}

	session.History = append(session.History, entry)
	if len(session.History) > chat.Settings.MaxMessages {
		session.History = session.History[len(session.History)-chat.Settings.MaxMessages:]
	}

	// Build message list from history
	var messages []gpt.Message
	if session.SystemPrompt != "" {
		messages = append(messages, gpt.Message{Role: "system", Content: session.SystemPrompt})
	}
	for _, e := range session.History {
		messages = append(messages, gpt.Message{Role: e.Prompt.Role, Content: e.Prompt.Content})
		if e.Response != (storage.Message{}) {
			messages = append(messages, gpt.Message{Role: e.Response.Role, Content: e.Response.Content})
		}
	}

	response := fallbackResponse
	payload, err := s.GptClient.CallGPT(messages, session.Model, session.Temperature)
	s.ErrorLog.LogError(err)

	if err == nil && payload != nil && len(payload.Choices) > 0 {
		response = strings.TrimSpace(fmt.Sprintf("%v", payload.Choices[0].Message.Content))
	}

	entry.Response = storage.Message{Role: "assistant", Content: response}
	return response
}

// GPTCommand sends a one-shot system+user prompt pair to GPT and returns
// the response text. Unlike ChatCompletion, it does not touch chat history.
func (s *GPTService) GPTCommand(model string, systemPrompt, userPrompt string) (string, error) {
	payload, err := s.GptClient.CallGPT([]gpt.Message{
		{Role: "system", Content: []gpt.Content{{Type: gpt.TypeText, Text: systemPrompt}}},
		{Role: "user", Content: []gpt.Content{{Type: gpt.TypeText, Text: userPrompt}}},
	}, model, 0.6)

	if err != nil {
		return "", err
	}

	response := fallbackResponse
	if len(payload.Choices) > 0 {
		response = strings.TrimSpace(fmt.Sprintf("%v", payload.Choices[0].Message.Content))
	}
	return response, nil
}

// ReadChatLog returns the last N lines from a chat's log file.
func (s *GPTService) ReadChatLog(chatID int64, count int) ([]string, error) {
	return util.ReadLastLines(fmt.Sprintf("log/%d.log", chatID), count)
}

// GenerateImage creates an image from a prompt and returns the URL along
// with an AI-enhanced caption.
func (s *GPTService) GenerateImage(model string, prompt string) (imageURL, caption string, err error) {
	imageURL, err = s.GptClient.GenerateImage(prompt, gpt.ImageSize1024)
	if err != nil {
		return "", "", err
	}

	caption = prompt
	payload, err := s.GptClient.CallGPT([]gpt.Message{
		{Role: "system", Content: "You are an assistant that generates natural language description (prompt) for an artificial intelligence (AI) that generates images"},
		{Role: "user", Content: fmt.Sprintf("Please improve this prompt: \"%s\". Answer with improved prompt only. Keep prompt at most 200 characters long. Your prompt must be in one sentence.", prompt)},
	}, model, 0.7)
	if err == nil && payload != nil && len(payload.Choices) > 0 {
		caption = strings.TrimSpace(fmt.Sprintf("%v", payload.Choices[0].Message.Content))
	}

	return imageURL, caption, nil
}
