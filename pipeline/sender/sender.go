package sender

// MessageSender is the delivery interface consumed by the pipeline.
// Concrete implementations live in the transport layer (e.g. api/telegram.Bot).
type MessageSender interface {
	Reply(chatID int64, replyTo int, text string)
	ReplyMarkdown(chatID int64, replyTo int, text string, isMarkdown bool)
	SendImageData(chatID int64, data []byte, caption string) error
	AudioUpload(chatID int64, bytes []byte) error

	// ReplyWithButtons sends a text reply carrying an inline keyboard.
	ReplyWithButtons(chatID int64, replyTo int, text string, markdown bool, buttons [][]Button) error
	// EditMessage replaces the text and inline keyboard of an existing message
	// (used when a button tap should update the message in place).
	EditMessage(chatID int64, messageID int, text string, markdown bool, buttons [][]Button) error
	// AnswerCallback acknowledges a button tap so Telegram stops the loading spinner.
	AnswerCallback(callbackID string, text string) error
}

// ResponseSender delivers a list of Response items via a MessageSender.
// It inspects which fields are set on each Response and picks the
// appropriate delivery method (text, image, audio).
type ResponseSender struct {
	Bot     MessageSender
	OnError func(error) // called when SendImageData / AudioUpload fails; may be nil
}

// Send delivers every response in order.
func (s *ResponseSender) Send(chatID int64, messageID int, responses []Response) {
	for _, r := range responses {
		switch {
		case len(r.Audio) > 0:
			if err := s.Bot.AudioUpload(chatID, r.Audio); err != nil && s.OnError != nil {
				s.OnError(err)
			}
		case len(r.ImageData) > 0:
			if err := s.Bot.SendImageData(chatID, r.ImageData, r.Caption); err != nil && s.OnError != nil {
				s.OnError(err)
			}
		case r.Text != "":
			if len(r.Buttons) > 0 {
				if err := s.Bot.ReplyWithButtons(chatID, messageID, r.Text, r.Markdown, r.Buttons); err != nil && s.OnError != nil {
					s.OnError(err)
				}
			} else {
				s.Bot.ReplyMarkdown(chatID, messageID, r.Text, r.Markdown)
			}
		}
	}
}

// Edit delivers responses produced by a button tap: it acknowledges the
// callback and edits the originating message in place (text + keyboard) instead
// of sending new messages. Only text responses are edited; any image/audio
// responses are ignored (button-driven commands are expected to be text-only).
func (s *ResponseSender) Edit(chatID int64, messageID int, callbackID string, responses []Response) {
	_ = s.Bot.AnswerCallback(callbackID, "")
	for _, r := range responses {
		if r.Text == "" {
			continue
		}
		if err := s.Bot.EditMessage(chatID, messageID, r.Text, r.Markdown, r.Buttons); err != nil && s.OnError != nil {
			s.OnError(err)
		}
	}
}
