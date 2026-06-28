package commands

import (
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"strings"
)

// CommandMenu is the top-level button launcher (the entry point that replaces
// the old text /help dump). Sections deep-link into the self-contained hubs
// (/list, /settings) or into a sub-menu rendered here.
//
// Sub-routes:
//
//	""       → main launcher
//	"info"   → info sub-menu (usage / context / history), back → main
//	"tools"  → text-tools hint sub-menu, back → main
type CommandMenu struct{}

func (c *CommandMenu) Name() string        { return "menu" }
func (c *CommandMenu) Description() string { return "Главное меню (кнопки)." }
func (c *CommandMenu) IsAdmin() bool       { return false }

func (c *CommandMenu) Execute(ctx *pipeline.RequestContext, ch *chat.Chat) []sender.Response {
	switch strings.TrimSpace(ctx.CommandArgs) {
	case "imagine":
		ch.PendingInput = "imagine"
		return forceReplyPrompt("Опишите картинку для генерации:")
	case "translate":
		ch.PendingInput = "translate"
		return forceReplyPrompt("Пришлите: <язык> <текст>\nНапример: en Привет, как дела")
	case "tech_translate":
		ch.PendingInput = "tech_translate"
		return forceReplyPrompt("Пришлите текст для перевода на технический английский:")
	case "enhance":
		ch.PendingInput = "enhance"
		return forceReplyPrompt("Пришлите текст для улучшения:")
	case "grammar":
		ch.PendingInput = "grammar"
		return forceReplyPrompt("Пришлите текст для проверки грамматики:")
	case "analyze":
		ch.PendingInput = "analyze"
		return forceReplyPrompt("Пришлите: <число сообщений> <промпт>")
	case "summarize":
		ch.PendingInput = "summarize"
		return forceReplyPrompt("Пришлите число сообщений для суммаризации (по умолчанию 50):")
	case "info":
		return []sender.Response{{
			Text: "ℹ️ Инфо",
			Buttons: [][]sender.Button{
				{{Text: "📊 Использование", Data: "usage:"}},
				{{Text: "📐 Контекст", Data: "context:"}},
				{{Text: "🕓 История", Data: "history:"}},
				{{Text: "📖 Команды", Data: "help:list"}},
				{{Text: "⬅ Назад", Data: "menu:"}},
			},
		}}
	case "tools":
		return []sender.Response{{
			Text: "🛠 Инструменты\n\nВыберите — бот попросит ввод.",
			Buttons: [][]sender.Button{
				{{Text: "🎨 Картинка", Data: "menu:imagine"}},
				{{Text: "🌐 Перевод", Data: "menu:translate"}},
				{{Text: "💻 Тех. перевод", Data: "menu:tech_translate"}},
				{{Text: "✨ Улучшить текст", Data: "menu:enhance"}},
				{{Text: "📝 Грамматика", Data: "menu:grammar"}},
				{{Text: "🔍 Анализ", Data: "menu:analyze"}},
				{{Text: "📄 Суммаризация", Data: "menu:summarize"}},
				{{Text: "⬅ Назад", Data: "menu:"}},
			},
		}}
	}

	return mainMenuView()
}

// mainMenuView is the top-level launcher keyboard, shared by /menu and /help.
func mainMenuView() []sender.Response {
	return []sender.Response{{
		Text: "📋 Меню",
		Buttons: [][]sender.Button{
			{{Text: "🗂 Сессии", Data: "list:"}},
			{{Text: "⚙️ Настройки", Data: "settings:"}},
			{{Text: "🛠 Инструменты", Data: "menu:tools"}},
			{{Text: "ℹ️ Инфо", Data: "menu:info"}},
		},
	}}
}
