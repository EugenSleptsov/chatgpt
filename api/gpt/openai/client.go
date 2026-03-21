package openai

import (
	"GPTBot/api/gpt"
	"GPTBot/api/logger"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// RequestResponsesPayload is the OpenAI Responses API request body.
type RequestResponsesPayload struct {
	Model        string        `json:"model"`
	Instructions string        `json:"instructions,omitempty"`
	Input        []gpt.Message `json:"input"`
	Tools        []gpt.Tool    `json:"tools,omitempty"`
	Store        bool          `json:"store"` // false = do not persist on OpenAI servers
}

// defaultTools lists tools enabled on every CallGPT request.
var defaultTools = []gpt.Tool{
	{Type: "web_search"},
}

// Client implements gpt.Client using the OpenAI Responses API.
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

func (c *Client) CallGPT(chatConversation []gpt.Message, aimodel string, instructions string, tools ...gpt.Tool) (*gpt.Response, error) {
	outerAiModel := gpt.ResolveAPIName(aimodel)

	allTools := make([]gpt.Tool, 0, len(defaultTools)+len(tools))
	allTools = append(allTools, defaultTools...)
	allTools = append(allTools, tools...)

	requestPayload := RequestResponsesPayload{
		Model:        outerAiModel,
		Instructions: instructions,
		Input:        chatConversation,
		Tools:        allTools,
		Store:        false,
	}

	jsonPayload, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, err
	}

	resp, err := c.Transport.Post("https://api.openai.com/v1/responses", "application/json", jsonPayload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	c.Log.Logf("Responses / HTTP status: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var responsePayload gpt.Response
	if err = json.Unmarshal(body, &responsePayload); err != nil {
		return nil, err
	}

	return &responsePayload, nil
}
