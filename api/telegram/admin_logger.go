package telegram

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// AdminLogger is a thin Telegram bot that sends administrative
// notifications to a dedicated chat (typically the bot owner).
// It uses the shared Transport layer, but may have a different
// bot token from the main Bot.
type AdminLogger struct {
	adminID   int64
	transport *Transport
}

// NewAdminLogger creates an AdminLogger that sends messages to adminID
// using the provided bot token. The token may differ from the main bot.
func NewAdminLogger(token string, adminID int64) (*AdminLogger, error) {
	transport, err := NewTransport(token)
	if err != nil {
		return nil, err
	}

	return &AdminLogger{
		adminID:   adminID,
		transport: transport,
	}, nil
}

// Log sends a text message to the admin chat, splitting it into
// 4096-rune chunks if necessary (Telegram message limit).
func (a *AdminLogger) Log(message string) error {
	const maxLen = 4096
	runes := []rune(message)

	for len(runes) > maxLen {
		if err := a.send(string(runes[:maxLen])); err != nil {
			return err
		}
		runes = runes[maxLen:]
	}

	return a.send(string(runes))
}

func (a *AdminLogger) send(text string) error {
	msg := tgbotapi.NewMessage(a.adminID, text)
	_, err := a.transport.Send(msg)
	return err
}
