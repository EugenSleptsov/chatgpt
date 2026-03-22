// Package service contains transport-agnostic business logic.
// GPTService wraps GPT API calls (responses, images, logs)
// and knows nothing about Telegram or any other transport.
package service

import (
	"GPTBot/api/gpt"
	"GPTBot/storage"
	"GPTBot/util"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

const fallbackResponse = "Произошла ошибка с получением ответа, пожалуйста, попробуйте позднее"

// GPTService encapsulates GPT business logic (responses, image generation, logs),
// independent of any transport layer (Telegram, etc.).
type GPTService struct {
	GptClient gpt.Client
	LogDir    string
}

// ChatResult holds the full output of a GPT call, including tool-call results.
type ChatResult struct {
	Text      string        // assistant text reply
	Images    []ImageResult // generated images (from generate_image tool calls)
	Audio     []byte        // generated audio  (from generate_voice tool call)
	AudioText string        // text that was synthesized (for history storage)
}

// ImageResult is one image produced by a generate_image tool call.
type ImageResult struct {
	URL     string
	Caption string
}

// buildHistoryContent composes the full assistant turn for storage in chat history.
// Text reply, generated images (as caption) and audio (as transcript) are all included
// so GPT has full context of what was produced in previous turns.
func buildHistoryContent(r *ChatResult) string {
	var parts []string
	if r.Text != "" {
		parts = append(parts, r.Text)
	}
	for _, img := range r.Images {
		parts = append(parts, fmt.Sprintf("[Сгенерирована картинка: %s]", img.Caption))
	}
	if r.AudioText != "" {
		parts = append(parts, fmt.Sprintf("[Сгенерировано аудио: «%s»]", r.AudioText))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n")
}

// chatTools are the function tools sent with every chat completion request.
// GPT decides which (if any) to call based on the user's message.
var chatTools = []gpt.Tool{
	{
		Type:        "function",
		Name:        "generate_image",
		Description: "Generate an image based on a text description. Call when the user asks to draw, create, or generate a picture/image.",
		Parameters: &gpt.FunctionParameters{
			Type: "object",
			Properties: map[string]gpt.ParameterProperty{
				"prompt": {Type: "string", Description: "Image description for the image generator (English preferred)"},
			},
			Required: []string{"prompt"},
		},
	},
	{
		Type:        "function",
		Name:        "generate_voice",
		Description: "Convert text to a voice/audio message. Call when the user asks to record voice, speak out loud, or create audio.",
		Parameters: &gpt.FunctionParameters{
			Type: "object",
			Properties: map[string]gpt.ParameterProperty{
				"text": {Type: "string", Description: "The text to convert to speech"},
			},
			Required: []string{"text"},
		},
	},
}

// ChatCompletion appends a user message to chat history, sends the
// conversation context to GPT (with function tools), runs the full tool
// loop (execute → send results back → model finalises), and returns
// the complete result.
func (s *GPTService) ChatCompletion(chat *storage.Chat, userText string) (*ChatResult, error) {
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

	payload, err := s.GptClient.CallGPT(messages, session.Model, session.SystemPrompt, chatTools...)
	if err != nil {
		log.Printf("[ChatCompletion] GPT error: %v", err)
		result := &ChatResult{Text: fallbackResponse}
		entry.Response = storage.Message{Role: "assistant", Content: fallbackResponse}
		return result, err
	}

	result, err := s.toolLoop(payload, session.Model, session.SystemPrompt)
	entry.Response = storage.Message{Role: "assistant", Content: buildHistoryContent(result)}
	return result, err
}

// maxToolIterations limits how many tool-call rounds the model can do
// in a single request to prevent infinite loops.
const maxToolIterations = 5

// toolLoop implements the correct function-calling cycle:
//  1. model returns function_call(s)
//  2. we execute each function
//  3. we send function_call_output(s) back (with the same call_id)
//  4. model produces a final answer (or more tool calls → repeat)
func (s *GPTService) toolLoop(response *gpt.Response, model, systemPrompt string) (*ChatResult, error) {
	result := &ChatResult{}

	for i := 0; i < maxToolIterations; i++ {
		calls := response.ToolCalls()
		if len(calls) == 0 {
			result.Text = strings.TrimSpace(response.OutputText())
			return result, nil
		}

		log.Printf("[ToolLoop] iteration %d: %d tool call(s)", i+1, len(calls))

		outputs := make([]gpt.ToolCallOutput, 0, len(calls))
		for _, tc := range calls {
			output := s.executeSingleToolCall(tc, result)
			outputs = append(outputs, gpt.NewToolCallOutput(tc.ID, output))
		}

		var err error
		response, err = s.GptClient.ContinueWithToolOutputs(
			response.ID, outputs, model, systemPrompt, chatTools...,
		)
		if err != nil {
			log.Printf("[ToolLoop] error continuing response: %v", err)
			if result.Text == "" {
				result.Text = fallbackResponse
			}
			return result, err
		}
	}

	// Max iterations reached — take whatever text the last response has.
	log.Printf("[ToolLoop] max iterations (%d) reached", maxToolIterations)
	result.Text = strings.TrimSpace(response.OutputText())
	if result.Text == "" {
		result.Text = fallbackResponse
	}
	return result, nil
}

// toolResult is the JSON structure sent back to the model as function output.
type toolResult struct {
	Status  string `json:"status"`
	URL     string `json:"url,omitempty"`
	Caption string `json:"caption,omitempty"`
	Text    string `json:"text,omitempty"`
	Error   string `json:"error,omitempty"`
}

func marshalToolResult(r toolResult) string {
	data, _ := json.Marshal(r)
	return string(data)
}

// executeSingleToolCall runs one tool call and returns the JSON output
// string to send back to the model. Side-effects (images, audio) are
// accumulated into result.
func (s *GPTService) executeSingleToolCall(tc gpt.ToolCall, result *ChatResult) string {
	log.Printf("[ToolCall] %s(%v)", tc.Name, tc.Args)
	switch tc.Name {
	case "generate_image":
		return s.executeImageToolCall(tc, result)
	case "generate_voice":
		return s.executeVoiceToolCall(tc, result)
	default:
		log.Printf("[ToolCall] unknown tool: %s", tc.Name)
		return marshalToolResult(toolResult{Status: "error", Error: "unknown tool: " + tc.Name})
	}
}

func (s *GPTService) executeImageToolCall(tc gpt.ToolCall, result *ChatResult) string {
	prompt := tc.Args["prompt"]
	if prompt == "" {
		return marshalToolResult(toolResult{Status: "error", Error: "empty prompt"})
	}
	imageURL, caption, err := s.GenerateImage(gpt.ImageEnhanceTierID, prompt)
	if err != nil {
		log.Printf("[ToolCall] generate_image error: %v", err)
		return marshalToolResult(toolResult{Status: "error", Error: err.Error()})
	}
	result.Images = append(result.Images, ImageResult{URL: imageURL, Caption: caption})
	return marshalToolResult(toolResult{Status: "success", URL: imageURL, Caption: caption})
}

func (s *GPTService) executeVoiceToolCall(tc gpt.ToolCall, result *ChatResult) string {
	text := tc.Args["text"]
	if text == "" {
		text = result.Text
	}
	if text == "" {
		return marshalToolResult(toolResult{Status: "error", Error: "no text available for voice synthesis"})
	}
	audio, err := s.GenerateVoice(text)
	if err != nil {
		log.Printf("[ToolCall] generate_voice error: %v", err)
		return marshalToolResult(toolResult{Status: "error", Error: err.Error()})
	}
	result.Audio = audio
	result.AudioText = text
	return marshalToolResult(toolResult{Status: "success", Text: text})
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
	return s.GptClient.GenerateVoice(text, gpt.VoiceModelHD, gpt.VoiceOnyx)
}

// --- Group chat methods ---

// LogGroupMessage stores a group participant's message in history without triggering a GPT reply.
// Content is attributed: "Author: text".
func (s *GPTService) LogGroupMessage(chat *storage.Chat, author, text string) {
	session := chat.ActiveSession()
	session.History = append(session.History, &storage.ConversationEntry{
		Prompt: storage.Message{Role: "user", Content: fmt.Sprintf("%s: %s", author, text)},
	})
	if len(session.History) > chat.Settings.MaxMessages {
		session.History = session.History[len(session.History)-chat.Settings.MaxMessages:]
	}
}

// LogGroupPhoto stores a photo placeholder in history — no analysis, no GPT call.
func (s *GPTService) LogGroupPhoto(chat *storage.Chat, author string, description string) {
	s.LogGroupMessage(chat, author, fmt.Sprintf("[Фото] %s", description))
}

// LogGroupSticker stores a sticker placeholder in history.
func (s *GPTService) LogGroupSticker(chat *storage.Chat, author, emoji string) {
	text := "[Стикер]"
	if emoji != "" {
		text = fmt.Sprintf("[Стикер: %s]", emoji)
	}
	s.LogGroupMessage(chat, author, text)
}

// LogBotResponse attaches the bot's reply to the last history entry.
// Call after LogGroupMessage when the bot answers a specific message out-of-band
// (e.g. image analysis that doesn't go through ReplyFromGroupHistory).
func (s *GPTService) LogBotResponse(chat *storage.Chat, text string) {
	session := chat.ActiveSession()
	if len(session.History) == 0 {
		return
	}
	last := session.History[len(session.History)-1]
	if last.Response == (storage.Message{}) {
		last.Response = storage.Message{Role: "assistant", Content: text}
	}
}

// ReplyFromGroupHistory sends the full group history to GPT (with function
// tools), runs the full tool loop, attaches the assistant response to the
// last history entry and returns the full ChatResult.
func (s *GPTService) ReplyFromGroupHistory(chat *storage.Chat) (*ChatResult, error) {
	session := chat.ActiveSession()
	if len(session.History) == 0 {
		return &ChatResult{Text: fallbackResponse}, nil
	}

	messages := storage.ToGPTMessages(session.History)
	payload, err := s.GptClient.CallGPT(messages, session.Model, session.SystemPrompt, chatTools...)
	if err != nil {
		log.Printf("[ReplyFromGroupHistory] GPT error: %v", err)
		return &ChatResult{Text: fallbackResponse}, err
	}

	result, err := s.toolLoop(payload, session.Model, session.SystemPrompt)

	session.History[len(session.History)-1].Response = storage.Message{
		Role: "assistant", Content: buildHistoryContent(result),
	}
	return result, err
}

// autoReplyCheckInstructions is the system prompt for the YES/NO auto-reply decision.
const autoReplyCheckInstructions = `You are an active participant in a group chat. Your name is the bot mentioned in the conversation.
Look at the recent messages and decide if you should respond.
Reply YES if:
- someone is talking to you or about you (even without @mention — e.g. "бот, ответь")
- there is a factual question no one answered
- you can add something genuinely useful or funny
- someone shared news/link and no one commented
Reply NO only if:
- the conversation is purely between other people and doesn't need your input
- your last message already addressed the topic and nothing new was added
Respond with exactly one word: YES or NO.`

// silenceThreshold is the number of consecutive messages without a bot response
// after which the bot forcibly joins the conversation.
const silenceThreshold = 20

// messagesSinceLastReply counts how many entries at the tail of history
// have no bot response. Used to force auto-reply after prolonged silence.
func messagesSinceLastReply(history []*storage.ConversationEntry) int {
	count := 0
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Response != (storage.Message{}) {
			break
		}
		count++
	}
	return count
}

// ShouldAutoReply asks GPT (using the last few messages) whether the bot should
// proactively join the group conversation.
// Forces YES if the bot has been silent for silenceThreshold messages.
// Returns (decision, reason, error).
func (s *GPTService) ShouldAutoReply(chat *storage.Chat) (bool, string, error) {
	session := chat.ActiveSession()

	// Forced reply after prolonged silence
	silent := messagesSinceLastReply(session.History)
	if silent >= silenceThreshold {
		return true, fmt.Sprintf("молчал %d сообщений, принудительный ответ", silent), nil
	}

	const lookback = 10
	history := session.History
	if len(history) > lookback {
		history = history[len(history)-lookback:]
	}

	messages := storage.ToGPTMessages(history)
	if len(messages) == 0 {
		return false, "история пуста", nil
	}

	payload, err := s.GptClient.CallGPT(messages, session.Model, autoReplyCheckInstructions)
	if err != nil {
		return false, "ошибка GPT", err
	}
	answer := strings.TrimSpace(payload.OutputText())
	yes := strings.HasPrefix(strings.ToUpper(answer), "YES")
	return yes, answer, nil
}
