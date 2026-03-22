package mock

import (
	"GPTBot/domain/ai"
	"fmt"
)

// Client implements ai.Client with static stub responses.
// Use it for tests and local development without real API keys.
type Client struct{}

// NewClient creates a new MockClient.
func NewClient() *Client {
	return &Client{}
}

func (m *Client) CallGPT(chatConversation []ai.Message, aimodel string, instructions string, tools ...ai.Tool) (*ai.Response, error) {
	return &ai.Response{
		ID:     "mock-id",
		Object: "response",
		Output: []ai.ResponseOutputItem{
			{
				Type: "message",
				ID:   "mock-msg-id",
				Role: "assistant",
				Content: []ai.ResponseOutputContent{
					{Type: "output_text", Text: fmt.Sprintf("[mock] echo: model=%s", aimodel)},
				},
			},
		},
		Usage: ai.ResponseUsage{
			InputTokens:  10,
			OutputTokens: 5,
			TotalTokens:  15,
		},
	}, nil
}

func (m *Client) ContinueWithToolOutputs(_ string, outputs []ai.ToolCallOutput, aimodel string, _ string, _ ...ai.Tool) (*ai.Response, error) {
	summary := fmt.Sprintf("[mock] tool outputs received: %d", len(outputs))
	return &ai.Response{
		ID:     "mock-continue-id",
		Object: "response",
		Output: []ai.ResponseOutputItem{
			{
				Type: "message",
				ID:   "mock-continue-msg",
				Role: "assistant",
				Content: []ai.ResponseOutputContent{
					{Type: "output_text", Text: summary},
				},
			},
		},
		Usage: ai.ResponseUsage{InputTokens: 5, OutputTokens: 5, TotalTokens: 10},
	}, nil
}

func (m *Client) GenerateImage(prompt string, size string) (string, error) {
	return fmt.Sprintf("https://mock.example.com/image?prompt=%s&size=%s", prompt, size), nil
}

func (m *Client) GenerateVoice(inputText string, _ string, _ string) ([]byte, error) {
	return []byte("mock-audio:" + inputText), nil
}

func (m *Client) TranscribeAudio(_ []byte) (string, error) {
	return "mock transcription", nil
}

// compile-time check: Client implements ai.Client
var _ ai.Client = (*Client)(nil)
