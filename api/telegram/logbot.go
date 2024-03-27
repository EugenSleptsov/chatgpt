package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type LogBot struct {
	Token string
	api   *tgbotapi.BotAPI
}

func NewLogBot(token string) (*LogBot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	return &LogBot{
		Token: token,
		api:   api,
	}, nil
}

func (bot *LogBot) SendMessage(chatId int64, text string) error {
	// split long messages
	for len(text) > 4096 {
		err := bot.message(chatId, text[:4096])
		if err != nil {
			return err
		}

		text = text[4096:]
	}

	return bot.message(chatId, text)
}

func (bot *LogBot) message(chatId int64, text string) error {
	msg := tgbotapi.NewMessage(chatId, text)
	_, err := bot.api.Send(msg)
	return err
}
