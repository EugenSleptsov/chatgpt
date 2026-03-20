// Package service contains transport-agnostic business logic.
// GPTService wraps GPT API calls (responses, images, logs)
// and knows nothing about Telegram or any other transport.
package service

import (
	"GPTBot/api/gpt"
	"GPTBot/storage"
	"GPTBot/util"
	"fmt"
	"strings"
)

const fallbackResponse = "Произошла ошибка с получением ответа, пожалуйста, попробуйте позднее"

// GPTService encapsulates GPT business logic (responses, image generation, logs),
// independent of any transport layer (Telegram, etc.).
type GPTService struct {
	GptClient gpt.Client
	LogDir    string
}

// ChatCompletion appends a user message to chat history, sends the
// conversation context to GPT, stores the assistant response in history
// and returns it. On GPT failure returns a fallback message and the error.
func (s *GPTService) ChatCompletion(chat *storage.Chat, userText string) (string, error) {
	session := chat.ActiveSession()

	entry := &storage.ConversationEntry{
		Prompt: storage.Message{Role: "user", Content: userText},
	}

	session.History = append(session.History, entry)
	if len(session.History) > chat.Settings.MaxMessages {
		session.History = session.History[len(session.History)-chat.Settings.MaxMessages:]
	}

	// Build message list from history — system prompt goes as instructions, not as a message.
	messages := storage.ToGPTMessages(session.History)

	response := fallbackResponse
	payload, err := s.GptClient.CallGPT(messages, session.Model, session.SystemPrompt)

	if err == nil {
		if text := strings.TrimSpace(payload.OutputText()); text != "" {
			response = text
		}
	}

	entry.Response = storage.Message{Role: "assistant", Content: response}
	return response, err
}

// GPTCommand sends a one-shot system+user prompt pair to GPT and returns
// the response text. Unlike ChatCompletion, it does not touch chat history.
func (s *GPTService) GPTCommand(model string, systemPrompt, userPrompt string) (string, error) {
	payload, err := s.GptClient.CallGPT([]gpt.Message{
		{Role: "user", Content: []gpt.Content{{Type: gpt.TypeInputText, Text: userPrompt}}},
	}, model, systemPrompt)

	if err != nil {
		return "", err
	}

	if text := strings.TrimSpace(payload.OutputText()); text != "" {
		return text, nil
	}
	return fallbackResponse, nil
}

// ReadChatLog returns the last N lines from a chat's log file.
func (s *GPTService) ReadChatLog(chatID int64, count int) ([]string, error) {
	return util.ReadLastLines(fmt.Sprintf("%s/%d.log", s.LogDir, chatID), count)
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
		{Role: "user", Content: fmt.Sprintf("Please improve this prompt: \"%s\". Answer with improved prompt only. Keep prompt at most 200 characters long. Your prompt must be in one sentence.", prompt)},
	}, model, "You are an assistant that generates natural language description (prompt) for an artificial intelligence (AI) that generates images")
	if err == nil {
		if text := strings.TrimSpace(payload.OutputText()); text != "" {
			caption = text
		}
	}

	return imageURL, caption, nil
}

// AnalyzeImage sends an image URL with a prompt to GPT Vision and returns the response.
func (s *GPTService) AnalyzeImage(imageURL, prompt string) (string, error) {
	messages := []gpt.Message{
		{Role: "user", Content: []gpt.Content{
			{Type: gpt.TypeInputText, Text: prompt},
			{Type: gpt.TypeInputImage, ImageUrl: imageURL},
		}},
	}

	payload, err := s.GptClient.CallGPT(messages, gpt.VisionTierID, "")
	if err != nil {
		return "", err
	}

	if text := strings.TrimSpace(payload.OutputText()); text != "" {
		return text, nil
	}
	return fallbackResponse, nil
}

// TranscribeAudio delegates audio transcription to the GPT provider.
func (s *GPTService) TranscribeAudio(audioContent []byte) (string, error) {
	return s.GptClient.TranscribeAudio(audioContent)
}

// GenerateVoice delegates text-to-speech to the GPT provider.
func (s *GPTService) GenerateVoice(text string) ([]byte, error) {
	return s.GptClient.GenerateVoice(text, gpt.VoiceModel, gpt.VoiceOnyx)
}
