package handler

// Response represents one unit of bot output.
// The worker inspects which fields are set and delivers accordingly.
type Response struct {
	Text     string // text reply
	ImageURL string // image URL to send
	Caption  string // image caption
	Audio    []byte // audio data to upload
	Markdown bool   // format text as markdown/HTML
}
