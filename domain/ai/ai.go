// Package ai contains the core domain contracts for AI provider integration.
// The Client interface and all request/response types are defined here.
// Concrete implementations live in integration/ai/openai.
package ai

import (
	"encoding/base64"
	"encoding/json"
	"log"
)

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
	VoiceModelHD = "tts-1-hd"
)

// --- Voice name constants ---

const (
	VoiceOnyx = "onyx"
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

// Message is a single input or output message in a conversation.
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
// Type can be "message", "function_call", "web_search_call",
// "image_generation_call", etc.
type ResponseOutputItem struct {
	Type    string                  `json:"type"`
	ID      string                  `json:"id"`
	Role    string                  `json:"role,omitempty"`
	Content []ResponseOutputContent `json:"content,omitempty"`
	// Function call fields (type == "function_call")
	Name      string `json:"name,omitempty"`
	CallID    string `json:"call_id,omitempty"`
	Arguments string `json:"arguments,omitempty"`
	// Image generation fields (type == "image_generation_call")
	Result string `json:"result,omitempty"` // base64-encoded PNG
}

// ResponseInputTokensDetails provides a breakdown of input token consumption.
type ResponseInputTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

// ResponseOutputTokensDetails provides a breakdown of output token consumption.
type ResponseOutputTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens"`
}

// ResponseUsage tracks token consumption for a Responses API request.
type ResponseUsage struct {
	InputTokens         int                          `json:"input_tokens"`
	OutputTokens        int                          `json:"output_tokens"`
	TotalTokens         int                          `json:"total_tokens"`
	InputTokensDetails  *ResponseInputTokensDetails  `json:"input_tokens_details,omitempty"`
	OutputTokensDetails *ResponseOutputTokensDetails `json:"output_tokens_details,omitempty"`
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
			if err := json.Unmarshal([]byte(item.Arguments), &args); err != nil {
				log.Printf("[ToolCalls] failed to decode arguments for %s: %v (raw: %s)", item.Name, err, item.Arguments)
				args = make(map[string]string)
			}
			calls = append(calls, ToolCall{ID: item.CallID, Name: item.Name, Args: args})
		}
	}
	return calls
}

// ImageResults returns decoded PNG data for every image_generation_call in the output.
func (r *Response) ImageResults() [][]byte {
	if r == nil {
		return nil
	}
	var images [][]byte
	for _, item := range r.Output {
		if item.Type == "image_generation_call" && item.Result != "" {
			data, err := base64.StdEncoding.DecodeString(item.Result)
			if err != nil {
				log.Printf("[ImageResults] failed to decode base64 image: %v", err)
				continue
			}
			images = append(images, data)
		}
	}
	return images
}

// ToolCall represents a single function invocation made by the model.
type ToolCall struct {
	ID   string
	Name string
	Args map[string]string
}

// ToolCallOutput is the result of executing a tool call, sent back to the
// model so it can formulate a final answer based on the actual outcome.
type ToolCallOutput struct {
	Type   string `json:"type"` // always "function_call_output"
	CallID string `json:"call_id"`
	Output string `json:"output"`
}

// NewToolCallOutput creates a ToolCallOutput with the correct type tag.
func NewToolCallOutput(callID, output string) ToolCallOutput {
	return ToolCallOutput{Type: "function_call_output", CallID: callID, Output: output}
}

// --- Client interface ---

// Client is the public API of the ai package.
// Implement this interface to add alternative providers (e.g. Anthropic, local LLM).
type Client interface {
	CallGPT(chatConversation []Message, aimodel string, instructions string, tools ...Tool) (*Response, error)
	// ContinueWithToolOutputs sends tool execution results back to the model
	// so it can produce a final answer. previousResponseID links to the response
	// that contained the original function_call items.
	ContinueWithToolOutputs(previousResponseID string, outputs []ToolCallOutput, aimodel string, instructions string, tools ...Tool) (*Response, error)
	GenerateImage(prompt string, size string) (string, error)
	GenerateVoice(inputText string, voiceModel, voiceVoice string) ([]byte, error)
	TranscribeAudio(audioContent []byte) (string, error)
}
