package commands

import (
	"GPTBot/domain/ai"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
	"strconv"
	"strings"
)

type CommandSessionList struct{}

func (c *CommandSessionList) Name() string {
	return "list"
}

func (c *CommandSessionList) Description() string {
	return "Показывает список сессий (чатов) с кнопками выбора."
}

func (c *CommandSessionList) IsAdmin() bool {
	return false
}

// sessionsPerPage caps how many session buttons fit on one keyboard page.
const sessionsPerPage = 6

// Execute renders the session picker. CommandArgs (typed empty, or a button
// payload) selects the behaviour:
//
//	""        → first page (or the page holding the active session)
//	"<n>"     → navigate to page n (from a "list:<n>" nav button)
//	"use:<id>"→ switch to session <id>, then show the page holding it
//	"new"     → create a new session, switch to it, show the page holding it
func (c *CommandSessionList) Execute(ctx *pipeline.RequestContext, chat *chat.Chat) []sender.Response {
	args := strings.TrimSpace(ctx.CommandArgs)

	page := 0
	switch {
	case strings.HasPrefix(args, "use:"):
		if id, err := strconv.Atoi(args[len("use:"):]); err == nil {
			if s := chat.FindSession(id); s != nil {
				chat.ActiveSessionID = s.ID
			}
		}
		page = sessionPageOf(chat, chat.ActiveSessionID)
	case args == "new":
		s := chat.AddSession("untitled")
		chat.ActiveSessionID = s.ID
		page = sessionPageOf(chat, chat.ActiveSessionID)
	case args == "del":
		return sessionDeleteView(chat)
	case args != "":
		page, _ = strconv.Atoi(args)
	default:
		page = sessionPageOf(chat, chat.ActiveSessionID)
	}

	return sessionListView(chat, page)
}

// sessionDeleteView renders a picker of sessions to delete: one button per
// session (callback "remove:<id>", which opens a delete confirmation) plus a
// back button to the list.
func sessionDeleteView(chat *chat.Chat) []sender.Response {
	rows := make([][]sender.Button, 0, len(chat.Sessions)+1)
	for _, s := range chat.Sessions {
		rows = append(rows, []sender.Button{{
			Text: fmt.Sprintf("🗑 #%d — %s", s.ID, s.Topic),
			Data: fmt.Sprintf("remove:%d", s.ID),
		}})
	}
	rows = append(rows, []sender.Button{{Text: "⬅ Назад", Data: "list:"}})
	return []sender.Response{{Text: "Выберите сессию для удаления:", Buttons: rows}}
}

// sessionPageOf returns the page index that contains the session with the given
// ID (0 if not found).
func sessionPageOf(chat *chat.Chat, id int) int {
	for i, s := range chat.Sessions {
		if s.ID == id {
			return i / sessionsPerPage
		}
	}
	return 0
}

// sessionListView renders one page of the session list: a text summary plus one
// button per session (callback "list:use:<id>"), a "new session" button
// (callback "list:new"), and, when there is more than one page, a navigation
// row (callback "list:<page>").
func sessionListView(chat *chat.Chat, page int) []sender.Response {
	total := len(chat.Sessions)
	pages := (total + sessionsPerPage - 1) / sessionsPerPage
	if pages == 0 {
		pages = 1
	}
	if page < 0 {
		page = 0
	}
	if page >= pages {
		page = pages - 1
	}

	start := page * sessionsPerPage
	end := start + sessionsPerPage
	if end > total {
		end = total
	}

	var sb strings.Builder
	sb.WriteString("📋 Сессии:\n\n")

	rows := make([][]sender.Button, 0, end-start+1)
	for _, s := range chat.Sessions[start:end] {
		marker := "  "
		btnPrefix := ""
		if s.ID == chat.ActiveSessionID {
			marker, btnPrefix = "▶ ", "▶ "
		}
		modelLabel := s.Model
		if tier := ai.FindTier(s.Model); tier != nil {
			modelLabel = tier.Label
		}
		sb.WriteString(fmt.Sprintf("%s#%d — %s [%s, %d сообщ.]\n", marker, s.ID, s.Topic, modelLabel, len(s.History)))

		label := fmt.Sprintf("%s#%d — %s", btnPrefix, s.ID, s.Topic)
		rows = append(rows, []sender.Button{{Text: label, Data: fmt.Sprintf("list:use:%d", s.ID)}})
	}
	sb.WriteString(fmt.Sprintf("\nАктивная: #%d", chat.ActiveSessionID))

	actions := []sender.Button{{Text: "➕ Сессия", Data: "list:new"}}
	if total > 1 {
		actions = append(actions, sender.Button{Text: "🗑 Удалить", Data: "list:del"})
	}
	rows = append(rows, actions)

	if pages > 1 {
		var nav []sender.Button
		if page > 0 {
			nav = append(nav, sender.Button{Text: "◀", Data: fmt.Sprintf("list:%d", page-1)})
		}
		nav = append(nav, sender.Button{Text: fmt.Sprintf("%d/%d", page+1, pages), Data: fmt.Sprintf("list:%d", page)})
		if page < pages-1 {
			nav = append(nav, sender.Button{Text: "▶", Data: fmt.Sprintf("list:%d", page+1)})
		}
		rows = append(rows, nav)
	}

	rows = append(rows, []sender.Button{{Text: "⬅ Меню", Data: "menu:"}})

	return []sender.Response{{Text: sb.String(), Buttons: rows}}
}
