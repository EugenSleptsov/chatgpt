package main

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/util"
	byteslib "bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

const audioEndpoint = "https://api.openai.com/v1/audio/transcriptions"
const voiceEndpoint = "https://api.openai.com/v1/audio/speech"
const audioModel = "whisper-1"
const voiceModel = "tts-1"
const voiceModelHD = "tts-1-hd"

const (
	voiceAlloy   = "alloy"
	voiceEcho    = "echo"
	voiceFable   = "fable"
	voiceOnyx    = "onyx"
	voiceNova    = "nova"
	voiceShimmer = "shimmer"
)

type TranscriptionResponse struct {
	Text string `json:"text"`
}

func processAudio(bot *telegram.Bot, gptClient *gpt.GPTClient, fileID string) (string, error) {
	// Download the voice message file
	file, err := bot.GetFile(fileID)
	if err != nil {
		return "", fmt.Errorf("error getting file: %w", err)
	}

	// Download the audio file content
	audioURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", bot.Token, file.FilePath)
	audioContent, err := util.DownloadFile(audioURL)
	if err != nil {
		return "", fmt.Errorf("error downloading audio file: %w", err)
	}

	return transcribeAudio(gptClient.ApiKey, audioContent)
}

func transcribeAudio(apiKey string, audioContent []byte) (string, error) {
	// Create a new multipart writer
	body := &byteslib.Buffer{}
	writer := multipart.NewWriter(body)

	// Add the audio file to the request body
	part, err := writer.CreateFormFile("file", "audio.ogg") // Assuming Telegram voice messages are in Ogg format
	if err != nil {
		return "", fmt.Errorf("error creating form file: %w", err)
	}
	part.Write(audioContent)

	// Add other form fields (e.g., model name)
	writer.WriteField("model", audioModel)

	// Close the multipart writer
	err = writer.Close()
	if err != nil {
		return "", fmt.Errorf("error closing writer: %w", err)
	}

	// Create the HTTP request
	request, err := http.NewRequest("POST", audioEndpoint, body)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	// Set headers, including the OpenAI API key
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Authorization", "Bearer "+apiKey)

	// Send the HTTP request
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("error sending request: %w", err)
	}
	defer response.Body.Close()

	// Read and print the response body
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	var transcriptionResponse TranscriptionResponse
	err = json.Unmarshal(responseBody, &transcriptionResponse)
	if err != nil {
		return "", fmt.Errorf("Error parsing JSON response: %v", err)
	}

	return transcriptionResponse.Text, nil
}

func processVoice(bot *telegram.Bot, gptClient *gpt.GPTClient, chatID int64, inputText string) error {
	payload := fmt.Sprintf(`{"model": "%s", "voice": "%s", "input": "%s"}`, voiceModel, voiceOnyx, inputText)

	request, err := http.NewRequest("POST", voiceEndpoint, strings.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+gptClient.ApiKey)

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	bytes, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	return bot.AudioUpload(chatID, bytes)
}
