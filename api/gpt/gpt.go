package gpt

// --- Content type constants ---

const (
	TypeText     = "text"
	TypeImageUrl = "image_url"
)

// --- Image size constants (used by GenerateImage) ---

const (
	ImageSize256  = "256x256"
	ImageSize512  = "512x512"
	ImageSize1024 = "1024x1024"
)

// --- Voice model constants ---

const (
	VoiceModel   = "tts-1"
	VoiceModelHD = "tts-1-hd"
)

// --- Voice name constants ---

const (
	VoiceAlloy   = "alloy"
	VoiceEcho    = "echo"
	VoiceFable   = "fable"
	VoiceOnyx    = "onyx"
	VoiceNova    = "nova"
	VoiceShimmer = "shimmer"
)

// --- Message types ---

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

// --- Response types ---

// CompletionChoice represents a single choice in a chat completion response.
type CompletionChoice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// CompletionUsage tracks token consumption for a completion request.
type CompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// CompletionResponse is the provider-agnostic response structure for chat completions.
type CompletionResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int                `json:"created"`
	Choices []CompletionChoice `json:"choices"`
	Usage   CompletionUsage    `json:"usage"`
}

// --- Client interface ---

// Client is the public API of the gpt package.
// Implement this interface to add alternative providers (e.g. Anthropic, local LLM).
type Client interface {
	CallGPT(chatConversation []Message, aimodel string, temperature float32) (*CompletionResponse, error)
	GenerateImage(prompt string, size string) (string, error)
	GenerateVoice(inputText string, voiceModel, voiceVoice string) ([]byte, error)
	TranscribeAudio(audioContent []byte) (string, error)
}
