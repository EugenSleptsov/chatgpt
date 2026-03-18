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

func (m *Client) CallGPT(chatConversation []gpt.Message, aimodel string, temperature float32) (*gpt.CompletionResponse, error) {
	return &gpt.CompletionResponse{
		ID:      "mock-id",
		Object:  "chat.completion",
		Created: 0,
		Choices: []gpt.CompletionChoice{
			{
				Index:        0,
				Message:      gpt.Message{Role: "assistant", Content: fmt.Sprintf("[mock] echo: model=%s", aimodel)},
				FinishReason: "stop",
			},
		},
		Usage: gpt.CompletionUsage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
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
