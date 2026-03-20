package adminlog

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type TelegramAdminLogger struct {
	Token   string
	AdminId int64
	api     *tgbotapi.BotAPI
}

func NewTelegramAdminLogger(token string, adminId int64) (*TelegramAdminLogger, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	return &TelegramAdminLogger{
		Token:   token,
		AdminId: adminId,
		api:     api,
	}, nil
}

func (bot *TelegramAdminLogger) Log(message string) error {
	const maxLen = 4096
	runes := []rune(message)

	for len(runes) > maxLen {
		err := bot.message(string(runes[:maxLen]))
		if err != nil {
			return err
		}
		runes = runes[maxLen:]
	}

	return bot.message(string(runes))
}

func (bot *TelegramAdminLogger) message(text string) error {
	msg := tgbotapi.NewMessage(bot.AdminId, text)
	_, err := bot.api.Send(msg)
	return err
}
