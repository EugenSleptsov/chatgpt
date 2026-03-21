package handler

import "GPTBot/api/telegram"

// ResponseSender delivers a list of Response items to Telegram.
// It inspects which fields are set on each Response and picks the
// appropriate delivery method (text, image, audio).
type ResponseSender struct {
	Bot     telegram.BotAPI
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
		case r.ImageURL != "":
			if err := s.Bot.SendImage(chatID, r.ImageURL, r.Caption); err != nil && s.OnError != nil {
				s.OnError(err)
			}
		case r.Text != "":
			s.Bot.ReplyMarkdown(chatID, messageID, r.Text, r.Markdown)
		}
	}
}
