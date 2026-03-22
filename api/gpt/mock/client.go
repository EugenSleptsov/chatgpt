package mock

import (
	"GPTBot/api/gpt"
	"fmt"
)

// Client implements gpt.Client with static stub responses.
// Use it for tests and local development without real API keys.
type Client struct{}

// NewClient creates a new MockClient.
func NewClient() *Client {
	return &Client{}
}

func (m *Client) CallGPT(chatConversation []gpt.Message, aimodel string, instructions string, tools ...gpt.Tool) (*gpt.Response, error) {
	return &gpt.Response{
		ID:     "mock-id",
		Object: "response",
		Output: []gpt.ResponseOutputItem{
			{
				Type: "message",
				ID:   "mock-msg-id",
				Role: "assistant",
				Content: []gpt.ResponseOutputContent{
					{Type: "output_text", Text: fmt.Sprintf("[mock] echo: model=%s", aimodel)},
				},
			},
		},
		Usage: gpt.ResponseUsage{
			InputTokens:  10,
			OutputTokens: 5,
			TotalTokens:  15,
		},
	}, nil
}

func (m *Client) ContinueWithToolOutputs(_ string, outputs []gpt.ToolCallOutput, aimodel string, _ string, _ ...gpt.Tool) (*gpt.Response, error) {
	summary := fmt.Sprintf("[mock] tool outputs received: %d", len(outputs))
	return &gpt.Response{
		ID:     "mock-continue-id",
		Object: "response",
		Output: []gpt.ResponseOutputItem{
			{
				Type: "message",
				ID:   "mock-continue-msg",
				Role: "assistant",
				Content: []gpt.ResponseOutputContent{
					{Type: "output_text", Text: summary},
				},
			},
		},
		Usage: gpt.ResponseUsage{InputTokens: 20, OutputTokens: 10, TotalTokens: 30},
	}, nil
}

func (m *Client) GenerateImage(_ string, _ string) (string, error) {
	return "https://mock.example.com/image.png", nil
}

func (m *Client) GenerateVoice(_ string, _, _ string) ([]byte, error) {
	return []byte("mock-audio-bytes"), nil
}

func (m *Client) TranscribeAudio(_ []byte) (string, error) {
	return "[mock] transcribed text", nil
}

// compile-time check: Client implements gpt.Client
var _ gpt.Client = (*Client)(nil)
