package handler

// IntentType identifies what kind of action the bot should perform.
type IntentType int

const (
	IntentChat              IntentType = iota // private chat completion (with tools)
	IntentGroupReply                          // group reply (bot addressed, with tools)
	IntentGroupAutoReply                      // group auto-reply candidate (with tools)
	IntentAnalyzeImage                        // analyze an uploaded image via GPT Vision
	IntentEchoTranscription                   // echo voice transcription as text
)

func (t IntentType) String() string {
	switch t {
	case IntentChat:
		return "Chat"
	case IntentGroupReply:
		return "GroupReply"
	case IntentGroupAutoReply:
		return "GroupAutoReply"
	case IntentAnalyzeImage:
		return "AnalyzeImage"
	case IntentEchoTranscription:
		return "EchoTranscription"
	default:
		return "Unknown"
	}
}

// Intent represents a single resolved intention from a user message.
type Intent struct {
	Type IntentType
}
