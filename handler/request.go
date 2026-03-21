package handler

// MediaType describes the original input medium.
type MediaType int

const (
	MediaText    MediaType = iota // plain text message
	MediaVoice                    // voice/audio message (Text = transcription)
	MediaImage                    // photo with optional caption
	MediaSticker                  // sticker (usually handled inline, no Request)
)

// Request is the normalized representation of what the user sent.
// Handlers produce it; the processing layer consumes it.
type Request struct {
	Text          string    // normalized text (transcribed, caption, message text)
	ImageURL      string    // image URL, if any
	BotAddressed  bool      // bot was explicitly mentioned / replied-to
	IsForwarded   bool      // message was forwarded
	OriginalMedia MediaType // how the user originally sent it
	CommandName   string    // non-empty when the update is a bot command (e.g. "help")
	CommandArgs   string    // raw arguments after the command verb
}
