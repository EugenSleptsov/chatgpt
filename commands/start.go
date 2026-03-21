package commands

import (
	"GPTBot/api/telegram"
	"GPTBot/handler"
	"GPTBot/storage"
)

type CommandStart struct {
	*Deps
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

func (c *CommandStart) Execute(ctx *telegram.UpdateContext, chat *storage.Chat) []handler.Response {
	return reply("Здравствуйте! Я чатбот-помощник, и я здесь, чтобы помочь вам с любыми вопросами или задачами. Просто напишите ваш вопрос или запрос, и я сделаю все возможное, чтобы помочь вам! Для справки наберите /help.")
}
