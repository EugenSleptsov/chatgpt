package openai

import (
	"GPTBot/api/gpt"
	"encoding/json"
	"fmt"
	"io"
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
	ModelDalle2 = "dall-e-2"
	ModelDalle3 = "dall-e-3"
)

func (c *Client) GenerateImage(prompt string, size string) (string, error) {
	jsonPayload, err := json.Marshal(RequestImagePayload{
		Prompt: prompt,
		Size:   getImageSize(size),
		Model:  ModelDalle3,
	})
	if err != nil {
		return "", err
	}

	resp, err := c.Transport.Post("https://api.openai.com/v1/images/generations", "application/json", jsonPayload)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	c.Log.Logf("Image / HTTP status: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

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
		c.Log.Logf("Empty data array in response: %s", string(body))

		if responseData.Error.Message != "" && responseData.Error.Code == "content_policy_violation" {
			return "", fmt.Errorf("content policy violation: %s", responseData.Error.Message)
		}

		return "", fmt.Errorf("empty data array in response")
	}

	return responseData.Data[0].URL, nil
}

func getImageSize(size string) string {
	switch size {
	case gpt.ImageSize256, gpt.ImageSize512, gpt.ImageSize1024:
		return size
	default:
		return gpt.ImageSize1024
	}
}
