package gpt

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

// Tier represents a user-facing model option in /model command.
// Users pick a stable tier name; the underlying API model can be swapped at any time.
type Tier struct {
	ID       string // stored in chat settings (e.g. "basic")
	Label    string // display name shown to users (e.g. "gpt-basic")
	Desc     string // human-readable description
	APIModel string // actual OpenAI API model name (e.g. "gpt-5.4-nano")
}

// Tiers lists available user-facing tiers.
// To upgrade models — change APIModel here, nothing else.
var Tiers = []Tier{
	{ID: "basic", Label: "gpt-basic", Desc: "Экономичная, для простых задач", APIModel: "gpt-5.4-nano"},
	{ID: "fast", Label: "gpt-fast", Desc: "Быстрая, для кодинга и агентов", APIModel: "gpt-5.4-mini"},
	{ID: "premium", Label: "gpt-premium", Desc: "Максимальное качество рассуждений", APIModel: "gpt-5.4"},
}

const (
	DefaultTierID      = "basic"   // default tier for new chats
	VisionTierID       = "premium" // tier used for image analysis
	ImageEnhanceTierID = "basic"   // tier used for image prompt enhancement
)

// legacyTierMap maps old model IDs (stored in existing chats) to current tier IDs.
var legacyTierMap = map[string]string{
	// GPT-3.x
	"gpt-3": "basic", "gpt-3.5-turbo": "basic", "gpt-3.5-turbo-1106": "basic",
	"gpt-3.5-turbo-16k": "basic", "gpt-316": "basic",
	// GPT-4.x
	"gpt-4": "basic", "gpt-4-turbo": "fast", "gpt-4-turbo-preview": "fast",
	"gpt-4-vision-preview": "premium", "gpt-4o": "fast", "gpt-4o-mini": "basic",
	// GPT-4.1
	"4.1-nano": "basic", "4.1-mini": "fast", "4.1": "premium",
	// GPT-5.x
	"gpt-5": "premium", "gpt-5-mini": "fast", "gpt-5-nano": "basic",
	"5.4-nano": "basic", "5.4-mini": "fast", "5.4": "premium", "5.4-pro": "premium",
	// o-series
	"o3-mini": "fast", "o3": "premium", "o4-mini": "fast",
}

// FindTier looks up a tier by ID or Label (handles legacy model IDs).
// Returns nil if not found.
func FindTier(id string) *Tier {
	if newID, ok := legacyTierMap[id]; ok {
		id = newID
	}
	for i := range Tiers {
		if Tiers[i].ID == id || Tiers[i].Label == id {
			return &Tiers[i]
		}
	}
	return nil
}

// DefaultTier returns the default tier.
func DefaultTier() Tier {
	return *FindTier(DefaultTierID)
}

// ResolveAPIName maps any tier/model ID (including legacy) to the actual OpenAI API name.
func ResolveAPIName(id string) string {
	if t := FindTier(id); t != nil {
		return t.APIModel
	}
	return DefaultTier().APIModel
}

// TierList returns a formatted string listing all available tiers.
func TierList() string {
	var result string
	for _, t := range Tiers {
		result += t.Label + " — " + t.Desc + "\n"
	}
	return result
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
	outerAiModel := ResolveAPIName(aimodel)

	requestPayload := RequestCompletionsPayload{
		Model:    outerAiModel,
		Messages: chatConversation,
		// Temperature: temperature,
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
		var req *http.Request
		req, err = http.NewRequest("POST", url, bytes.NewBuffer(payload))
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

