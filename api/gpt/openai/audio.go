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
	part.Write(audioContent)

	writer.WriteField("model", audioModel)

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
	payload := fmt.Sprintf(`{"model": "%s", "voice": "%s", "input": "%s"}`, voiceModel, voiceVoice, inputText)

	resp, err := c.Transport.Post(voiceEndpoint, "application/json", []byte(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
