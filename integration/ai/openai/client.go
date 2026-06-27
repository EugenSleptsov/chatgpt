package openai

import (
	"GPTBot/domain/ai"
	"GPTBot/infrastructure/logger"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// RequestResponsesPayload is the OpenAI Responses API request body.
type RequestResponsesPayload struct {
	Model        string        `json:"model"`
	Instructions string        `json:"instructions,omitempty"`
	Input        []ai.Message  `json:"input"`
	Tools        []ai.Tool     `json:"tools,omitempty"`
	Reasoning    *ai.Reasoning `json:"reasoning,omitempty"`
	Store        bool          `json:"store"`
}

// ContinueResponsesPayload continues a previous response with tool-call outputs.
type ContinueResponsesPayload struct {
	Model              string              `json:"model"`
	Instructions       string              `json:"instructions,omitempty"`
	PreviousResponseID string              `json:"previous_response_id"`
	Input              []ai.ToolCallOutput `json:"input"`
	Tools              []ai.Tool           `json:"tools,omitempty"`
	Reasoning          *ai.Reasoning       `json:"reasoning,omitempty"`
	Store              bool                `json:"store"`
}

var defaultTools []ai.Tool

// Client implements ai.Client using the OpenAI Responses API.
type Client struct {
	Transport Transport
	Log       logger.Log
}

// compile-time check: Client implements ai.Client
var _ ai.Client = (*Client)(nil)

func NewClient(apiKey string, log logger.Log) *Client {
	return &Client{Transport: NewHTTPTransport(apiKey), Log: log}
}

const responsesEndpoint = "https://api.openai.com/v1/responses"

func (c *Client) CallGPT(chatConversation []ai.Message, aimodel string, instructions string, tools ...ai.Tool) (*ai.Response, error) {
	outerAiModel := ResolveModel(aimodel)

	allTools := make([]ai.Tool, 0, len(defaultTools)+len(tools))
	allTools = append(allTools, defaultTools...)
	allTools = append(allTools, tools...)

	return c.postResponses(RequestResponsesPayload{
		Model:        outerAiModel,
		Instructions: instructions,
		Input:        chatConversation,
		Tools:        allTools,
		Reasoning:    ReasoningForTier(aimodel),
		Store:        true,
	})
}

func (c *Client) ContinueWithToolOutputs(previousResponseID string, outputs []ai.ToolCallOutput, aimodel string, instructions string, tools ...ai.Tool) (*ai.Response, error) {
	outerAiModel := ResolveModel(aimodel)

	allTools := make([]ai.Tool, 0, len(defaultTools)+len(tools))
	allTools = append(allTools, defaultTools...)
	allTools = append(allTools, tools...)

	return c.postResponses(ContinueResponsesPayload{
		Model:              outerAiModel,
		Instructions:       instructions,
		PreviousResponseID: previousResponseID,
		Input:              outputs,
		Tools:              allTools,
		Reasoning:          ReasoningForTier(aimodel),
		Store:              true,
	})
}

func (c *Client) postResponses(payload interface{}) (*ai.Response, error) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	resp, err := c.Transport.Post(responsesEndpoint, "application/json", jsonPayload)
	if err != nil {
		return nil, fmt.Errorf("transport error: %w", err)
	}
	defer resp.Body.Close()

	c.Log.Logf("GPT / HTTP status: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var response ai.Response
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &response, nil
}
