package gpt

import "encoding/json"

// --- Content type constants ---

const (
	TypeInputText  = "input_text"
	TypeInputImage = "input_image"
)

// --- Image size constants (used by GenerateImage) ---

const (
	ImageSize256  = "256x256"
	ImageSize512  = "512x512"
	ImageSize1024 = "1024x1024"
)

// --- Voice model constants ---

const (
	VoiceModel   = "tts-1"
	VoiceModelHD = "tts-1-hd"
)

// --- Voice name constants ---

const (
	VoiceAlloy   = "alloy"
	VoiceEcho    = "echo"
	VoiceFable   = "fable"
	VoiceOnyx    = "onyx"
	VoiceNova    = "nova"
	VoiceShimmer = "shimmer"
)

// --- Message types ---

// Content represents a single part of a multimodal message.
// For TypeInputText: set Text.
// For TypeInputImage: set ImageUrl (plain URL string).
type Content struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageUrl string `json:"image_url,omitempty"`
}

type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// --- Tool types ---

// Tool represents a tool available to the model.
// Built-in tools (like web_search) only need Type.
// Function tools also need Name, Description, and Parameters.
type Tool struct {
	Type        string              `json:"type"`                  // "web_search", "function"
	Name        string              `json:"name,omitempty"`        // function name
	Description string              `json:"description,omitempty"` // what the function does
	Parameters  *FunctionParameters `json:"parameters,omitempty"`  // JSON Schema
}

// FunctionParameters is the JSON Schema for a function tool's parameters.
type FunctionParameters struct {
	Type       string                       `json:"type"` // "object"
	Properties map[string]ParameterProperty `json:"properties"`
	Required   []string                     `json:"required,omitempty"`
}

// ParameterProperty describes a single parameter in the JSON Schema.
type ParameterProperty struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// --- Response types (Responses API) ---

// ResponseOutputContent is one content block inside an output message item.
type ResponseOutputContent struct {
	Type string `json:"type"` // "output_text"
	Text string `json:"text"`
}

// ResponseOutputItem is one element in the output array.
// Type can be "message", "function_call", "web_search_call", etc.
type ResponseOutputItem struct {
	Type    string                  `json:"type"`
	ID      string                  `json:"id"`
	Role    string                  `json:"role,omitempty"`
	Content []ResponseOutputContent `json:"content,omitempty"`
	// Function call fields (type == "function_call")
	Name      string `json:"name,omitempty"`
	CallID    string `json:"call_id,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// ResponseUsage tracks token consumption for a Responses API request.
type ResponseUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// Response is the provider-agnostic response structure for the Responses API.
type Response struct {
	ID     string               `json:"id"`
	Object string               `json:"object"`
	Output []ResponseOutputItem `json:"output"`
	Usage  ResponseUsage        `json:"usage"`
}

// OutputText returns the first assistant text from the response, or "".
func (r *Response) OutputText() string {
	if r == nil {
		return ""
	}
	for _, item := range r.Output {
		if item.Type == "message" && len(item.Content) > 0 {
			return item.Content[0].Text
		}
	}
	return ""
}

// ToolCalls returns all function_call items from the response output.
func (r *Response) ToolCalls() []ToolCall {
	if r == nil {
		return nil
	}
	var calls []ToolCall
	for _, item := range r.Output {
		if item.Type == "function_call" {
			var args map[string]string
			_ = json.Unmarshal([]byte(item.Arguments), &args)
			calls = append(calls, ToolCall{ID: item.CallID, Name: item.Name, Args: args})
		}
	}
	return calls
}

// ToolCall represents a single function invocation made by the model.
type ToolCall struct {
	ID   string
	Name string
	Args map[string]string
}

// --- Client interface ---

// Client is the public API of the gpt package.
// Implement this interface to add alternative providers (e.g. Anthropic, local LLM).
type Client interface {
	CallGPT(chatConversation []Message, aimodel string, instructions string, tools ...Tool) (*Response, error)
	GenerateImage(prompt string, size string) (string, error)
	GenerateVoice(inputText string, voiceModel, voiceVoice string) ([]byte, error)
	TranscribeAudio(audioContent []byte) (string, error)
}
