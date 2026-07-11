package commands

import (
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"strings"
)

// CommandHelp is a callback-first guide. Slash commands remain supported, but
// the help UI leads users to the same button hubs as the main menu.
type CommandHelp struct{}

func (c *CommandHelp) Name() string        { return "help" }
func (c *CommandHelp) Description() string { return "Интерактивная справка." }
func (c *CommandHelp) IsAdmin() bool       { return false }

func (c *CommandHelp) Execute(ctx *pipeline.RequestContext, _ *chat.Chat) []sender.Response {
	back := []sender.Button{{Text: "⬅ Справка", Data: "help:"}}
	switch strings.TrimSpace(ctx.CommandArgs) {
	case "sessions":
		return []sender.Response{{Text: "🗂 Сессии\n\nРазделяйте разные темы. Новую сессию можно начать чистой, полностью скопировать или перенести в неё компактное резюме текущего контекста.", Buttons: [][]sender.Button{{{Text: "Открыть сессии", Data: "list:"}}, back}}}
	case "memory":
		return []sender.Response{{Text: "🧠 Память и заметки\n\nПамять хранит долгосрочный контекст, а заметки — отдельные тематические записи и напоминания. Бот подтверждает сохранение только после успешной операции.", Buttons: [][]sender.Button{{{Text: "🧠 Память", Data: "memory:"}, {Text: "📒 Заметки", Data: "advisor:"}}, back}}}
	case "tools":
		return []sender.Response{{Text: "🛠 Инструменты\n\nПеревод, работа с текстом, анализ, суммаризация и генерация изображений запускаются кнопками — писать команды не обязательно.", Buttons: [][]sender.Button{{{Text: "Открыть инструменты", Data: "menu:tools"}}, back}}}
	case "settings":
		return []sender.Response{{Text: "⚙️ Настройки\n\nЗдесь меняются модель, форматирование, системный промпт, часовой пояс, память и другие параметры чата.", Buttons: [][]sender.Button{{{Text: "Открыть настройки", Data: "settings:"}}, back}}}
	case "chat":
		return []sender.Response{{Text: "💬 Как общаться\n\nПишите обычным текстом. В группах обратитесь к боту по имени, упомяните его или ответьте на его сообщение. Для действий используйте главное меню.", Buttons: [][]sender.Button{{{Text: "📋 Главное меню", Data: "menu:"}}, back}}}
	}
	return []sender.Response{{
		Text: "❓ Справка\n\nВыберите раздел:",
		Buttons: [][]sender.Button{
			{{Text: "🗂 Сессии", Data: "help:sessions"}},
			{{Text: "🧠 Память и заметки", Data: "help:memory"}},
			{{Text: "🛠 Инструменты", Data: "help:tools"}},
			{{Text: "⚙️ Настройки", Data: "help:settings"}},
			{{Text: "💬 Как общаться", Data: "help:chat"}},
			{{Text: "⬅ Меню", Data: "menu:"}},
		},
	}}
}
