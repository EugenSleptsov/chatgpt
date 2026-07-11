package telegram

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// Command represents a slash-command shown in Telegram's quick menu.
type Command string

const (
	CommandMenu Command = "menu"
	CommandNew  Command = "new"
	CommandList Command = "list"
	CommandHelp Command = "help"
)

var CommandDescriptions = map[Command]string{
	CommandMenu: "Главное меню с кнопками",
	CommandNew:  "Создать новую сессию",
	CommandList: "Открыть список сессий",
	CommandHelp: "Интерактивная справка",
}

// Keep the quick menu intentionally small: all secondary actions live in
// callback hubs opened from /menu.
var DefaultCommandList = []Command{CommandMenu, CommandNew, CommandList, CommandHelp}

// SetCommandList registers Telegram commands from config, falling back to the
// callback-first default when no supported configured commands remain.
func (botInstance *Bot) SetCommandList(rawCommandMenu []string) error {
	commandMenu := make([]Command, 0, len(rawCommandMenu))
	for _, command := range rawCommandMenu {
		if _, ok := CommandDescriptions[Command(command)]; ok {
			commandMenu = append(commandMenu, Command(command))
		}
	}
	if len(commandMenu) == 0 {
		commandMenu = DefaultCommandList
	}
	return botInstance.setCommandList(commandMenu...)
}

func (botInstance *Bot) setCommandList(commands ...Command) error {
	tgCommands := make([]tgbotapi.BotCommand, 0, len(commands))
	for _, command := range commands {
		tgCommands = append(tgCommands, tgbotapi.BotCommand{Command: string(command), Description: CommandDescriptions[command]})
	}
	_, err := botInstance.transport.Request(tgbotapi.NewSetMyCommands(tgCommands...))
	return err
}
