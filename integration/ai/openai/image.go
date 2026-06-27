package openai

import (
	"GPTBot/domain/ai"
	"encoding/base64"
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
		B64JSON string `json:"b64_json"`
	} `json:"data"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Type    string `json:"type"`
	}
}

// ModelGptImage is OpenAI's current image-generation model. Unlike DALL·E it
// returns base64-encoded PNG bytes (no hosted URL).
const ModelGptImage = "gpt-image-2"

// GenerateImage renders a prompt and returns the decoded PNG bytes.
func (c *Client) GenerateImage(prompt string, size string) ([]byte, error) {
	jsonPayload, err := json.Marshal(RequestImagePayload{
		Prompt: prompt,
		Size:   getImageSize(size),
		Model:  ModelGptImage,
	})
	if err != nil {
		return nil, err
	}

	resp, err := c.Transport.Post("https://api.openai.com/v1/images/generations", "application/json", jsonPayload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	c.Log.Logf("Image / HTTP status: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var responseData ResponseImagePayload
	if err = json.Unmarshal(body, &responseData); err != nil {
		return nil, err
	}

	if len(responseData.Data) == 0 || responseData.Data[0].B64JSON == "" {
		c.Log.Logf("Empty data array in response: %s", string(body))

		if responseData.Error.Message != "" && responseData.Error.Code == "content_policy_violation" {
			return nil, fmt.Errorf("content policy violation: %s", responseData.Error.Message)
		}

		return nil, fmt.Errorf("empty data array in response")
	}

	imageData, err := base64.StdEncoding.DecodeString(responseData.Data[0].B64JSON)
	if err != nil {
		return nil, fmt.Errorf("decode base64 image: %w", err)
	}
	return imageData, nil
}

func getImageSize(size string) string {
	switch size {
	case ai.ImageSize256, ai.ImageSize512, ai.ImageSize1024:
		return size
	default:
		return ai.ImageSize1024
	}
}
