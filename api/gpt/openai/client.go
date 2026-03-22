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
	Store        bool          `json:"store"` // must be true for previous_response_id to work in tool loop
}

// ContinueResponsesPayload continues a previous response with tool-call outputs.
type ContinueResponsesPayload struct {
	Model              string               `json:"model"`
	Instructions       string               `json:"instructions,omitempty"`
	PreviousResponseID string               `json:"previous_response_id"`
	Input              []gpt.ToolCallOutput `json:"input"`
	Tools              []gpt.Tool           `json:"tools,omitempty"`
	Store              bool                 `json:"store"`
}

// defaultTools lists built-in tools enabled on every CallGPT request.
var defaultTools = []gpt.Tool{
	{Type: "web_search"},
	{Type: "image_generation"},
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

const responsesEndpoint = "https://api.openai.com/v1/responses"

func (c *Client) CallGPT(chatConversation []gpt.Message, aimodel string, instructions string, tools ...gpt.Tool) (*gpt.Response, error) {
	outerAiModel := gpt.ResolveAPIName(aimodel)

	allTools := make([]gpt.Tool, 0, len(defaultTools)+len(tools))
	allTools = append(allTools, defaultTools...)
	allTools = append(allTools, tools...)

	return c.postResponses(RequestResponsesPayload{
		Model:        outerAiModel,
		Instructions: instructions,
		Input:        chatConversation,
		Tools:        allTools,
		Store:        true,
	})
}

func (c *Client) ContinueWithToolOutputs(previousResponseID string, outputs []gpt.ToolCallOutput, aimodel string, instructions string, tools ...gpt.Tool) (*gpt.Response, error) {
	outerAiModel := gpt.ResolveAPIName(aimodel)

	allTools := make([]gpt.Tool, 0, len(defaultTools)+len(tools))
	allTools = append(allTools, defaultTools...)
	allTools = append(allTools, tools...)

	return c.postResponses(ContinueResponsesPayload{
		Model:              outerAiModel,
		Instructions:       instructions,
		PreviousResponseID: previousResponseID,
		Input:              outputs,
		Tools:              allTools,
		Store:              true,
	})
}

// postResponses marshals a payload, POSTs it to the Responses API and
// decodes the response. Used by both CallGPT and ContinueWithToolOutputs.
func (c *Client) postResponses(payload interface{}) (*gpt.Response, error) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := c.Transport.Post(responsesEndpoint, "application/json", jsonPayload)
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
