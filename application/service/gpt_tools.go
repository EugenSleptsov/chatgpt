package service

import (
	"GPTBot/domain/ai"
	chatdomain "GPTBot/domain/chat"
	"encoding/json"
	"log"
	"strings"
)

// builtinTools are executed server-side by OpenAI: web_search returns text,
// image_generation returns base64 PNGs in the response output.
var builtinTools = []ai.Tool{
	{Type: "web_search"},
	{Type: "image_generation"},
}

// functionTools are executed client-side by executeSingleToolCall.
var functionTools = []ai.Tool{
	{
		Type:        "function",
		Name:        "generate_voice",
		Description: "Convert text to a voice/audio message. Call when the user asks to record voice, speak out loud, or create audio.",
		Parameters: &ai.FunctionParameters{
			Type: "object",
			Properties: map[string]ai.ParameterProperty{
				"text": {Type: "string", Description: "The text to convert to speech"},
			},
			Required: []string{"text"},
		},
	},
	{
		Type:        "function",
		Name:        "update_memory",
		Description: "Save or update a fact about the user/chat for future conversations. Call when you learn something worth remembering (name, preferences, context) or when the user explicitly asks to remember something. Each call adds one fact. Existing memory is shown in the system prompt.",
		Parameters: &ai.FunctionParameters{
			Type: "object",
			Properties: map[string]ai.ParameterProperty{
				"fact": {Type: "string", Description: "A single fact to remember, e.g. 'User prefers dark mode' or 'User's name is Alex'"},
			},
			Required: []string{"fact"},
		},
	},
}

// chatTools is the single tool set sent on every chat completion.
var chatTools = concatTools(builtinTools, functionTools)

func concatTools(a, b []ai.Tool) []ai.Tool {
	r := make([]ai.Tool, 0, len(a)+len(b))
	return append(append(r, a...), b...)
}

func toolNamesFromTools(tools []ai.Tool) []string {
	names := make([]string, 0, len(tools))
	for _, t := range tools {
		if t.Name != "" {
			names = append(names, t.Name)
		} else {
			names = append(names, t.Type)
		}
	}
	return names
}

// executeSingleToolCall runs one tool call and returns the JSON output string.
func (s *GPTService) executeSingleToolCall(tc ai.ToolCall, result *ChatResult, chat *chatdomain.Chat) string {
	log.Printf("[ToolCall] %s(%v)", tc.Name, tc.Args)
	switch tc.Name {
	case "generate_voice":
		return s.executeVoiceToolCall(tc, result)
	case "update_memory":
		return s.executeUpdateMemory(tc, chat)
	default:
		log.Printf("[ToolCall] unknown tool: %s", tc.Name)
		return marshalToolResult(toolResult{Status: "error", Error: "unknown tool: " + tc.Name})
	}
}

func (s *GPTService) executeVoiceToolCall(tc ai.ToolCall, result *ChatResult) string {
	text := tc.Args["text"]
	if text == "" {
		text = result.Text
	}
	if text == "" {
		return marshalToolResult(toolResult{Status: "error", Error: "no text available for voice synthesis"})
	}
	audio, err := s.GptClient.GenerateVoice(text, ai.VoiceModelHD, ai.VoiceOnyx)
	if err != nil {
		log.Printf("[ToolCall] generate_voice error: %v", err)
		return marshalToolResult(toolResult{Status: "error", Error: err.Error()})
	}
	result.Audio = audio
	result.AudioText = text
	return marshalToolResult(toolResult{Status: "success", Text: text})
}

func (s *GPTService) executeUpdateMemory(tc ai.ToolCall, chat *chatdomain.Chat) string {
	fact := strings.TrimSpace(tc.Args["fact"])
	if fact == "" {
		return marshalToolResult(toolResult{Status: "error", Error: "empty fact"})
	}
	AddMemory(chat, fact)
	return marshalToolResult(toolResult{Status: "success", Text: "Fact saved"})
}

// toolResult is the JSON structure returned by tool call handlers.
type toolResult struct {
	Status string `json:"status"`
	Text   string `json:"text,omitempty"`
	Error  string `json:"error,omitempty"`
}

func marshalToolResult(r toolResult) string {
	data, _ := json.Marshal(r)
	return string(data)
}
