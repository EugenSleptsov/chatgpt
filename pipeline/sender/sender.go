package sender

// MessageSender is the delivery interface consumed by the pipeline.
// Concrete implementations live in the transport layer (e.g. api/telegram.Bot).
type MessageSender interface {
	Reply(chatID int64, replyTo int, text string)
	ReplyMarkdown(chatID int64, replyTo int, text string, isMarkdown bool)
	SendImage(chatID int64, imageUrl string, caption string) error
	SendImageData(chatID int64, data []byte, caption string) error
	AudioUpload(chatID int64, bytes []byte) error
}

// ResponseSender delivers a list of Response items via a MessageSender.
// It inspects which fields are set on each Response and picks the
// appropriate delivery method (text, image, audio).
type ResponseSender struct {
	Bot     MessageSender
	OnError func(error) // called when SendImage / AudioUpload fails; may be nil
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
		case r.ImageURL != "":
			if err := s.Bot.SendImage(chatID, r.ImageURL, r.Caption); err != nil && s.OnError != nil {
				s.OnError(err)
			}
		case r.Text != "":
			s.Bot.ReplyMarkdown(chatID, messageID, r.Text, r.Markdown)
		}
	}
}
