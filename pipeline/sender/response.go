package sender

// Button is one inline-keyboard button. Data is the callback payload sent back
// when the button is tapped; by convention it is "<command>:<args>" so the tap
// can be routed through the same command registry as a typed command.
type Button struct {
	Text string
	Data string
}

// Response represents one unit of bot output.
// The worker inspects which fields are set and delivers accordingly.
type Response struct {
	Text      string     // text reply
	ImageData []byte     // image PNG bytes (/imagine command + built-in image_generation tool)
	Caption   string     // image caption
	Audio     []byte     // audio data to upload
	Markdown  bool       // format text as markdown/HTML
	Buttons   [][]Button // optional inline keyboard (rows of buttons); attached to text replies
}
