package gpt

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type RequestPayload struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float32   `json:"temperature,omitempty"`
}

type ResponsePayload struct {
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

func (gptClient *GPTClient) CallGPT35(chatConversation []Message, aimodel string, temperature float32) (*ResponsePayload, error) {
	url := "https://api.openai.com/v1/chat/completions"
	apiKey := gptClient.ApiKey

	payload := RequestPayload{
		Model:       aimodel,
		Messages:    chatConversation,
		Temperature: temperature,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	retries := 3
	var resp *http.Response

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
		time.Sleep(time.Duration(i+1) * time.Second) // Add a delay before retrying
	}

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Log the HTTP status code and status text
	log.Printf("HTTP status: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var responsePayload ResponsePayload
	err = json.Unmarshal(body, &responsePayload)
	if err != nil {
		return nil, err
	}

	return &responsePayload, nil
}
