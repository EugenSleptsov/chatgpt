package openai

import (
	"GPTBot/api/gpt"
	"GPTBot/api/logger"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// RequestCompletionsPayload is the OpenAI-specific request body.
type RequestCompletionsPayload struct {
	Model       string        `json:"model"`
	Messages    []gpt.Message `json:"messages"`
	Temperature float32       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

// Client implements gpt.Client using the OpenAI API.
type Client struct {
	Transport Transport
	Log       logger.Log
}

// compile-time check: Client implements gpt.Client
var _ gpt.Client = (*Client)(nil)

// NewClient creates a production OpenAI client.
func NewClient(apiKey string, log logger.Log) *Client {
	return &Client{Transport: NewHTTPTransport(apiKey), Log: log}
}

func (c *Client) CallGPT(chatConversation []gpt.Message, aimodel string, temperature float32) (*gpt.CompletionResponse, error) {
	outerAiModel := gpt.ResolveAPIName(aimodel)

	requestPayload := RequestCompletionsPayload{
		Model:    outerAiModel,
		Messages: chatConversation,
	}

	jsonPayload, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, err
	}

	resp, err := c.Transport.Post("https://api.openai.com/v1/chat/completions", "application/json", jsonPayload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	c.Log.Logf("Completions / HTTP status: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var responsePayload gpt.CompletionResponse
	if err = json.Unmarshal(body, &responsePayload); err != nil {
		return nil, err
	}

	return &responsePayload, nil
}
