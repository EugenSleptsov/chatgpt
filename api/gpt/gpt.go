package gpt

const (
	TypeText     = "text"
	TypeImageUrl = "image_url"
)

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

type Client interface {
	CallGPT(chatConversation []Message, aimodel string, temperature float32) (*ResponseCompletionsPayload, error)
	GenerateImage(prompt string, size string) (string, error)
	GenerateVoice(inputText string, voiceModel, voiceVoice string) ([]byte, error)
	TranscribeAudio(audioContent []byte) (string, error)
}
