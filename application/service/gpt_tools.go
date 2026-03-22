package service

import (
	"GPTBot/domain/ai"
	chatdomain "GPTBot/domain/chat"
	"encoding/json"
	"log"
	"strings"
)

// --- Tool proxy constants ---

const (
	proxySearchWeb   = "search_web"
	proxyCreateImage = "create_image"
)

var proxyBuiltinTools = []ai.Tool{
	{
		Type:        "function",
		Name:        proxySearchWeb,
		Description: "Search the internet for up-to-date information, recent events, real-time data, or any facts you are not sure about.",
		Parameters: &ai.FunctionParameters{
			Type: "object",
			Properties: map[string]ai.ParameterProperty{
				"query": {Type: "string", Description: "Search query"},
			},
			Required: []string{"query"},
		},
	},
	{
		Type:        "function",
		Name:        proxyCreateImage,
		Description: "Generate an image or picture from a text description. Use when the user asks to draw, create, imagine, or generate a visual.",
		Parameters: &ai.FunctionParameters{
			Type: "object",
			Properties: map[string]ai.ParameterProperty{
				"prompt": {Type: "string", Description: "Image description"},
			},
			Required: []string{"prompt"},
		},
	},
}

var realBuiltinTools = []ai.Tool{
	{Type: "web_search"},
	{Type: "image_generation"},
}

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

var (
	chatToolsLight = concatTools(proxyBuiltinTools, functionTools)
	chatToolsFull  = concatTools(realBuiltinTools, functionTools)
)

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

func hasProxyToolCalls(resp *ai.Response) bool {
	for _, tc := range resp.ToolCalls() {
		if tc.Name == proxySearchWeb || tc.Name == proxyCreateImage {
			return true
		}
	}
	return false
}

func (s *GPTService) upgradeProxyToBuiltin(resp *ai.Response, model, instructions string, chat *chatdomain.Chat) (*ai.Response, error) {
	calls := resp.ToolCalls()
	outputs := make([]ai.ToolCallOutput, 0, len(calls))
	for _, tc := range calls {
		switch tc.Name {
		case proxySearchWeb, proxyCreateImage:
			outputs = append(outputs, ai.NewToolCallOutput(tc.ID,
				marshalToolResult(toolResult{Status: "proceed", Text: "Use the built-in tool to complete this request."})))
		default:
			output := s.executeSingleToolCall(tc, &ChatResult{}, chat)
			outputs = append(outputs, ai.NewToolCallOutput(tc.ID, output))
		}
	}
	return s.GptClient.ContinueWithToolOutputs(resp.ID, outputs, model, instructions, chatToolsFull...)
}

func (s *GPTService) probeAndUpgrade(
	payload *ai.Response,
	model, instructions string,
	chat *chatdomain.Chat,
	caller string,
) (*ai.Response, []ai.Tool, TokenUsage, error) {
	var preflight TokenUsage

	if !hasProxyToolCalls(payload) {
		return payload, chatToolsLight, preflight, nil
	}

	triggered := triggeredBuiltinNames(payload)
	preflight.upgradePhase = "GPT + " + strings.Join(triggered, " + ")
	preflight.accumulate(extractUsage(payload, model, s.CostFn), "Probe (proxy tools)", toolNamesFromTools(chatToolsLight)...)
	log.Printf("[%s] proxy tool triggered (%s), upgrading to built-in tools", caller, strings.Join(triggered, ", "))

	upgraded, err := s.upgradeProxyToBuiltin(payload, model, instructions, chat)
	if err != nil {
		log.Printf("[%s] GPT upgrade error: %v", caller, err)
		return nil, chatToolsFull, preflight, err
	}
	return upgraded, chatToolsFull, preflight, nil
}

func triggeredBuiltinNames(resp *ai.Response) []string {
	seen := map[string]bool{}
	var names []string
	for _, tc := range resp.ToolCalls() {
		var realTool string
		switch tc.Name {
		case proxySearchWeb:
			realTool = "web_search"
		case proxyCreateImage:
			realTool = "image_generation"
		}
		if realTool != "" && !seen[realTool] {
			seen[realTool] = true
			names = append(names, realTool)
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
	s.Memory.Add(chat, fact)
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
