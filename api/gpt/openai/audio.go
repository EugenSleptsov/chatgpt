package openai

import (
	byteslib "bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
)

const audioEndpoint = "https://api.openai.com/v1/audio/transcriptions"
const voiceEndpoint = "https://api.openai.com/v1/audio/speech"
const audioModel = "whisper-1"

type TranscriptionResponse struct {
	Text string `json:"text"`
}

func (c *Client) TranscribeAudio(audioContent []byte) (string, error) {
	body := &byteslib.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "audio.ogg")
	if err != nil {
		return "", fmt.Errorf("error creating form file: %w", err)
	}
	if _, err = part.Write(audioContent); err != nil {
		return "", fmt.Errorf("error writing audio content: %w", err)
	}

	if err = writer.WriteField("model", audioModel); err != nil {
		return "", fmt.Errorf("error writing model field: %w", err)
	}

	if err = writer.Close(); err != nil {
		return "", fmt.Errorf("error closing writer: %w", err)
	}

	resp, err := c.Transport.Post(audioEndpoint, writer.FormDataContentType(), body.Bytes())
	if err != nil {
		return "", fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	var transcriptionResponse TranscriptionResponse
	if err = json.Unmarshal(responseBody, &transcriptionResponse); err != nil {
		return "", fmt.Errorf("error parsing JSON response: %w", err)
	}

	return transcriptionResponse.Text, nil
}

func (c *Client) GenerateVoice(inputText string, voiceModel, voiceVoice string) ([]byte, error) {
	type voiceRequest struct {
		Model string `json:"model"`
		Voice string `json:"voice"`
		Input string `json:"input"`
	}
	jsonPayload, err := json.Marshal(voiceRequest{
		Model: voiceModel,
		Voice: voiceVoice,
		Input: inputText,
	})
	if err != nil {
		return nil, fmt.Errorf("error marshaling voice request: %w", err)
	}

	resp, err := c.Transport.Post(voiceEndpoint, "application/json", jsonPayload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
