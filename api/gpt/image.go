package gpt

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type RequestImagePayload struct {
	Prompt string `json:"prompt"`
	Size   string `json:"size"`
	Model  string `json:"model"`
}

type ResponseImagePayload struct {
	Data []struct {
		URL string `json:"url"`
	} `json:"data"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Type    string `json:"type"`
	}
}

const (
	ImageSize256  = "256x256"
	ImageSize512  = "512x512"
	ImageSize1024 = "1024x1024"
	ModelDalle2   = "dall-e-2"
	ModelDalle3   = "dall-e-3"
)

func (gptClient *OpenAiGPTClient) GenerateImage(prompt string, size string) (string, error) {
	jsonPayload, err := json.Marshal(RequestImagePayload{
		Prompt: prompt,
		Size:   getImageSize(size),
		Model:  ModelDalle3,
	})
	if err != nil {
		return "", err
	}

	resp, err := gptClient.jsonRequest("https://api.openai.com/v1/images/generations", jsonPayload, 1)
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
		log.Printf("Empty data array in response: %s", string(body))

		if responseData.Error.Message != "" && responseData.Error.Code == "content_policy_violation" {
			return "", fmt.Errorf("content policy violation: %s", responseData.Error.Message)
		}

		return "", fmt.Errorf("empty data array in response")
	}

	return responseData.Data[0].URL, nil
}

func getImageSize(size string) string {
	switch size {
	case ImageSize256, ImageSize512, ImageSize1024:
		return size
	default:
		return ImageSize1024 // Default to 1024x1024 if size is invalid
	}
}
