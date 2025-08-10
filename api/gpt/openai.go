package gpt

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

// model names
const (
	ModelGPT3         = "gpt-3"
	ModelGPT3Turbo    = "gpt-3.5-turbo"
	ModelGPT3TurboX   = "gpt-3.5-turbo-1106"
	ModelGPT316k      = "gpt-3.5-turbo-16k"
	ModelGPT316k2     = "gpt-316"
	ModelGPT4         = "gpt-4"
	ModelGPT4Turbo    = "gpt-4-turbo"
	ModelGPT4Preview  = "gpt-4-turbo-preview"
	ModelGPT4Vision   = "gpt-4-vision-preview"
	ModelGPT4Omni     = "gpt-4o"
	ModelGPT4OmniMini = "gpt-4o-mini"
	ModelGPT5         = "gpt-5"
	ModelGPT5Mini     = "gpt-5-mini"
	ModelGPT5Nano     = "gpt-5-nano"
)

// outer model names
const (
	OuterModelGPT3     = "gpt-3.5-turbo"
	OuterModelGPT4mini = "gpt-4o-mini"
	OuterModelGPT4     = "gpt-4o"
	OuterModelGPT5     = "gpt-5"
	OuterModelGPT5mini = "gpt-5-mini"
	OuterModelGPT5nano = "gpt-5-nano"
)

var ModelMap = map[string]string{
	ModelGPT3:         OuterModelGPT5nano,
	ModelGPT3Turbo:    OuterModelGPT5nano,
	ModelGPT3TurboX:   OuterModelGPT5nano,
	ModelGPT316k:      OuterModelGPT5nano,
	ModelGPT316k2:     OuterModelGPT5nano,
	ModelGPT4:         OuterModelGPT5nano,
	ModelGPT4Turbo:    OuterModelGPT5nano,
	ModelGPT4Preview:  OuterModelGPT5nano,
	ModelGPT4Vision:   OuterModelGPT5nano,
	ModelGPT4Omni:     OuterModelGPT5nano,
	ModelGPT4OmniMini: OuterModelGPT5nano,
	ModelGPT5:         OuterModelGPT5,
	ModelGPT5Mini:     OuterModelGPT5nano,
}

type RequestCompletionsPayload struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float32   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

type ResponseCompletionsPayload struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Choices []struct {
		Index        int `json:"index"`
		Message      Message
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type OpenAiGPTClient struct {
	ApiKey string
}

func NewGPTClient(apiKey string) *OpenAiGPTClient {
	return &OpenAiGPTClient{ApiKey: apiKey}
}

func (gptClient *OpenAiGPTClient) CallGPT(chatConversation []Message, aimodel string, temperature float32) (*ResponseCompletionsPayload, error) {
	outerAiModel := MapModelName(aimodel)

	requestPayload := RequestCompletionsPayload{
		Model:    outerAiModel,
		Messages: chatConversation,
		// Temperature: temperature,
	}

	if aimodel == ModelGPT4Vision {
		requestPayload.MaxTokens = 4096
	}

	jsonPayload, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, err
	}

	retries := 3
	resp, err := gptClient.jsonRequest("https://api.openai.com/v1/chat/completions", jsonPayload, retries)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	log.Printf("Completions / HTTP status: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var responsePayload ResponseCompletionsPayload
	err = json.Unmarshal(body, &responsePayload)
	if err != nil {
		return nil, err
	}

	return &responsePayload, nil
}

func (gptClient *OpenAiGPTClient) jsonRequest(url string, jsonPayload []byte, retries int) (*http.Response, error) {
	return gptClient.httpRequest(url, "application/json", jsonPayload, retries)
}

func (gptClient *OpenAiGPTClient) httpRequest(url, contentType string, payload []byte, retries int) (*http.Response, error) {
	apiKey := gptClient.ApiKey

	var resp *http.Response
	var err error

	for i := 0; i < retries; i++ {
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Authorization", "Bearer "+apiKey)

		client := &http.Client{}
		resp, err = client.Do(req)
		if err == nil && resp.StatusCode == 200 {
			break
		}
		time.Sleep(time.Duration(i+1) * time.Second)
	}

	return resp, err
}

func MapModelName(modelName string) string {
	if mappedModel, exists := ModelMap[modelName]; exists {
		return mappedModel
	}
	return OuterModelGPT5nano // default model if not found
}
