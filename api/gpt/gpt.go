package gpt

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	TypeText     = "text"
	TypeImageUrl = "image_url"
)

type ImageUrl struct {
	Url    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

type Content struct {
	Type     string   `json:"type"`
	Text     string   `json:"text,omitempty"`
	ImageUrl ImageUrl `json:"image_url,omitempty"`
}

type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
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

type GPTClient struct {
	ApiKey string
}

// model names
const (
	ModelGPT3        = "gpt-3"
	ModelGPT3Turbo   = "gpt-3.5-turbo"
	ModelGPT3TurboX  = "gpt-3.5-turbo-1106"
	ModelGPT316k     = "gpt-3.5-turbo-16k"
	ModelGPT316k2    = "gpt-316"
	ModelGPT4        = "gpt-4"
	ModelGPT4Turbo   = "gpt-4-turbo"
	ModelGPT4Preview = "gpt-4-turbo-preview"
	ModelGPT4Vision  = "gpt-4-vision-preview"
	ModelGPT4Omni    = "gpt-4o"
)

// outer model names
const (
	OuterModelGPT3 = "gpt-3.5-turbo"
	OuterModelGPT4 = "gpt-4o"
)

var ModelMap = map[string]string{
	ModelGPT3:        OuterModelGPT3,
	ModelGPT3Turbo:   OuterModelGPT3,
	ModelGPT3TurboX:  OuterModelGPT3,
	ModelGPT316k:     OuterModelGPT3,
	ModelGPT316k2:    OuterModelGPT3,
	ModelGPT4:        OuterModelGPT4,
	ModelGPT4Turbo:   OuterModelGPT4,
	ModelGPT4Preview: OuterModelGPT4,
	ModelGPT4Vision:  OuterModelGPT4,
	ModelGPT4Omni:    OuterModelGPT4,
}

func NewGPTClient(apiKey string) *GPTClient {
	return &GPTClient{ApiKey: apiKey}
}

func (gptClient *GPTClient) CallGPT(chatConversation []Message, aimodel string, temperature float32) (*ResponseCompletionsPayload, error) {
	outerAiModel := ModelMap[aimodel]
	if outerAiModel == "" {
		outerAiModel = OuterModelGPT3
	}

	requestPayload := RequestCompletionsPayload{
		Model:       outerAiModel,
		Messages:    chatConversation,
		Temperature: temperature,
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

func (gptClient *GPTClient) jsonRequest(url string, jsonPayload []byte, retries int) (*http.Response, error) {
	return gptClient.httpRequest(url, "application/json", jsonPayload, retries)
}

func (gptClient *GPTClient) httpRequest(url, contentType string, payload []byte, retries int) (*http.Response, error) {
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
