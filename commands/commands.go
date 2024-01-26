package commands

import (
	"GPTBot/api/gpt"
	"GPTBot/api/telegram"
	"GPTBot/storage"
)

var CommandList = map[string]Command{
	"help": &CommandHelp{},
	"test": &CommandTest{},
}

type Command interface {
	Name() string
	Description() string
	Execute(bot *telegram.Bot, update telegram.Update, gptClient *gpt.GPTClient, chat *storage.Chat)
}
