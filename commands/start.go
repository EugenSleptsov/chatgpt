package commands

import (
	"GPTBot/api/telegram"
	"GPTBot/storage"
)

type CommandStart struct {
	TelegramBot *telegram.Bot
}

func (c *CommandStart) Name() string {
	return "start"
}

func (c *CommandStart) Description() string {
	return "Отправляет приветственное сообщение, описывающее цель бота."
}

func (c *CommandStart) IsAdmin() bool {
	return false
}

func (c *CommandStart) Execute(update telegram.Update, chat *storage.Chat) {
	c.TelegramBot.Reply(chat.ChatID, update.Message.MessageID, "Здравствуйте! Я чатбот-помощник, и я здесь, чтобы помочь вам с любыми вопросами или задачами. Просто напишите ваш вопрос или запрос, и я сделаю все возможное, чтобы помочь вам! Для справки наберите /help.")
}
