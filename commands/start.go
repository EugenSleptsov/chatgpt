package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
)

type CommandStart struct{}

func (c *CommandStart) Name() string {
	return "start"
}

func (c *CommandStart) Description() string {
	return "Start chat with bot"
}

func (c *CommandStart) Execute(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat) {
	bot.Reply(chat.ChatID, update.Message.MessageID, "Здравствуйте! Я чатбот-помощник, и я здесь, чтобы помочь вам с любыми вопросами или задачами. Просто напишите ваш вопрос или запрос, и я сделаю все возможное, чтобы помочь вам! Для справки наберите /help.")
}
