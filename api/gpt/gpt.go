package gpt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type RequestCompletionsPayload struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float32   `json:"temperature,omitempty"`
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

type RequestImagePayload struct {
	Prompt string `json:"prompt"`
	Size   string `json:"size"`
	Model  string `json:"model"`
}

type ResponseImagePayload struct {
	Data []struct {
		URL string `json:"url"`
	} `json:"data"`
}

const (
	ImageSize256     = "256x256"
	ImageSize512     = "512x512"
	ImageSize1024    = "1024x1024"
	ModelGPT3        = "gpt-3"
	ModelGPT3Turbo   = "gpt-3.5-turbo-1106"
	ModelGPT316k     = "gpt-3.5-turbo-16k"
	ModelGPT316k2    = "gpt-316"
	ModelGPT4        = "gpt-4"
	ModelGPT4Preview = "gpt-4-turbo-preview"
	ModelDalle2      = "dall-e-2"
	ModelDalle3      = "dall-e-3"
)

type GPTClient struct {
	ApiKey string
}

func NewGPTClient(apiKey string) *GPTClient {
	return &GPTClient{ApiKey: apiKey}
}

func (gptClient *GPTClient) CallGPT35(chatConversation []Message, aimodel string, temperature float32) (*ResponseCompletionsPayload, error) {
	jsonPayload, err := json.Marshal(RequestCompletionsPayload{
		Model:       aimodel,
		Messages:    chatConversation,
		Temperature: temperature,
	})
	if err != nil {
		return nil, err
	}

	retries := 3
	resp, err := gptClient.httpRequest("https://api.openai.com/v1/chat/completions", jsonPayload, retries)
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

func (gptClient *GPTClient) GenerateImage(prompt string, size string) (string, error) {
	jsonPayload, err := json.Marshal(RequestImagePayload{
		Prompt: prompt,
		Size:   getImageSize(size),
		Model:  ModelDalle3,
	})
	if err != nil {
		return "", err
	}

	resp, err := gptClient.httpRequest("https://api.openai.com/v1/images/generations", jsonPayload, 1)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	log.Printf("Image / HTTP status: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var responseData ResponseImagePayload
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		return "", err
	}

	if len(responseData.Data) == 0 {
		return "", fmt.Errorf("empty data array in response")
	}

	return responseData.Data[0].URL, nil
}

func (gptClient *GPTClient) httpRequest(url string, jsonPayload []byte, retries int) (*http.Response, error) {
	apiKey := gptClient.ApiKey

	var resp *http.Response
	var err error

	for i := 0; i < retries; i++ {
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/json")
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

func getImageSize(size string) string {
	switch size {
	case ImageSize256, ImageSize512, ImageSize1024:
		return size
	default:
		return ImageSize1024 // Default to 1024x1024 if size is invalid
	}
}
