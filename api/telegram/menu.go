package telegram

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// Command represents a slash-command name shown in the Telegram menu.
type Command string

const (
	CommandMenu      Command = "menu"
	CommandSettings  Command = "settings"
	CommandHelp      Command = "help"
	CommandHistory   Command = "history"
	CommandRollback  Command = "rollback"
	CommandClear     Command = "clear"
	CommandSummarize Command = "summarize"
)

// CommandDescriptions maps each command to a human-readable description.
var CommandDescriptions = map[Command]string{
	CommandMenu:      "Главное меню (кнопки)",
	CommandSettings:  "Настройки чата (кнопки)",
	CommandHelp:      "Справка по командам",
	CommandHistory:   "Показать историю переписки",
	CommandRollback:  "Отменить последнее сообщение",
	CommandClear:     "Очистить историю переписки",
	CommandSummarize: "Суммаризировать историю переписки",
}

// DefaultCommandList is the fallback menu when config.CommandMenu is empty.
var DefaultCommandList = []Command{
	CommandMenu,
	CommandSettings,
	CommandHelp,
	CommandHistory,
	CommandRollback,
	CommandClear,
	CommandSummarize,
}

// SetCommandList registers Telegram bot menu commands from raw config strings.
func (botInstance *Bot) SetCommandList(rawCommandMenu []string) {
	var commandMenu []Command
	for _, command := range rawCommandMenu {
		if _, ok := CommandDescriptions[Command(command)]; ok {
			commandMenu = append(commandMenu, Command(command))
		}
	}

	if len(commandMenu) > 0 {
		_ = botInstance.setCommandList(commandMenu...)
	} else {
		_ = botInstance.setCommandList(DefaultCommandList...)
	}
}

func (botInstance *Bot) setCommandList(commands ...Command) error {
	var tgCommands []tgbotapi.BotCommand
	for _, command := range commands {
		tgCommands = append(tgCommands, tgbotapi.BotCommand{Command: string(command), Description: CommandDescriptions[command]})
	}

	_, err := botInstance.transport.Request(tgbotapi.NewSetMyCommands(tgCommands...))
	return err
}
