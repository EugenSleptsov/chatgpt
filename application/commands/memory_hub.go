package commands

import (
	"GPTBot/application/service"
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
	"strconv"
	"strings"
)

// CommandMemory is the button hub for supermemory: browse the layered index
// (root nodes → children), read summaries, open the raw transcript behind a
// node and pin nodes against meta-compaction. Browsing works even when the
// setting is off — disabling hides memory from the model, not from the user.
//
// Sub-routes (callback data "memory:<sub>"):
//
//	""          → root node list, first page
//	"<n>"       → root node list, page n
//	"node:<id>" → node view (summary, children, source/pin/delete controls)
//	"src:<id>"  → raw transcript behind the node
//	"pin:<id>"  → toggle pin, re-render node view
//	"del:<id>"  → confirm node deletion (honours SkipDeleteConfirm)
//	"del:yes:<id>" → delete node (children become roots; archive untouched)
//	"toggle"    → flip the Supermemory setting, re-render list
type CommandMemory struct {
	Archive chat.Archive // raw transcript store (may be nil — source view degrades)
}

func (c *CommandMemory) Name() string        { return "memory" }
func (c *CommandMemory) Description() string { return "Суперпамять (кнопки)." }
func (c *CommandMemory) IsAdmin() bool       { return false }

// memoryNodesPerPage caps how many node buttons fit on one keyboard page.
const memoryNodesPerPage = 6

// memoryHubTextCap keeps node/source views inside Telegram's message limit.
const memoryHubTextCap = 3500

func (c *CommandMemory) Execute(ctx *pipeline.RequestContext, ch *chat.Chat) []sender.Response {
	args := strings.TrimSpace(ctx.CommandArgs)

	switch {
	case strings.HasPrefix(args, "node:"):
		if id, err := strconv.Atoi(args[len("node:"):]); err == nil {
			return memoryNodeView(ch, id)
		}
	case strings.HasPrefix(args, "src:"):
		if id, err := strconv.Atoi(args[len("src:"):]); err == nil {
			return c.memorySourceView(ch, id)
		}
	case strings.HasPrefix(args, "pin:"):
		if id, err := strconv.Atoi(args[len("pin:"):]); err == nil {
			if n := ch.FindMemoryNode(id); n != nil {
				n.Pinned = !n.Pinned
			}
			return memoryNodeView(ch, id)
		}
	case strings.HasPrefix(args, "del:yes:"):
		if id, err := strconv.Atoi(args[len("del:yes:"):]); err == nil {
			ch.RemoveMemoryNode(id)
		}
		return memoryIndexView(ch, 0)
	case strings.HasPrefix(args, "del:"):
		if id, err := strconv.Atoi(args[len("del:"):]); err == nil {
			return memoryNodeDeleteConfirm(ch, id)
		}
	case args == "toggle":
		ch.Settings.Supermemory = !ch.Settings.Supermemory
		return memoryIndexView(ch, 0)
	case args != "":
		if page, err := strconv.Atoi(args); err == nil {
			return memoryIndexView(ch, page)
		}
	}

	return memoryIndexView(ch, 0)
}

// memoryToggleButton renders the enable/disable control for the index view.
func memoryToggleButton(ch *chat.Chat) sender.Button {
	if ch.Settings.Supermemory {
		return sender.Button{Text: "⏸ Выключить", Data: "memory:toggle"}
	}
	return sender.Button{Text: "▶️ Включить", Data: "memory:toggle"}
}

// memoryIndexView renders one page of root nodes, newest first.
func memoryIndexView(ch *chat.Chat, page int) []sender.Response {
	status := "выключена (модель память не видит, данные сохранены)"
	if ch.Settings.Supermemory {
		status = "включена"
	}

	roots := ch.RootMemoryNodes()
	// Newest first: reverse the append-ordered roots.
	for i, j := 0, len(roots)-1; i < j; i, j = i+1, j-1 {
		roots[i], roots[j] = roots[j], roots[i]
	}

	if len(roots) == 0 {
		return []sender.Response{{
			Text: fmt.Sprintf("🧠 Суперпамять: %s\n\nУзлов пока нет — они появляются автоматически, когда старая переписка сжимается из контекста.", status),
			Buttons: [][]sender.Button{
				{memoryToggleButton(ch)},
				{{Text: "⬅ Меню", Data: "menu:"}},
			},
		}}
	}

	pages := (len(roots) + memoryNodesPerPage - 1) / memoryNodesPerPage
	if page < 0 {
		page = 0
	}
	if page >= pages {
		page = pages - 1
	}
	start := page * memoryNodesPerPage
	end := start + memoryNodesPerPage
	if end > len(roots) {
		end = len(roots)
	}

	rows := make([][]sender.Button, 0, end-start+3)
	for _, n := range roots[start:end] {
		label := truncate(n.Hook, 40)
		if n.Pinned {
			label = "📌 " + label
		}
		if len(n.Children) > 0 {
			label = "↓ " + label
		}
		rows = append(rows, []sender.Button{{
			Text: label,
			Data: fmt.Sprintf("memory:node:%d", n.ID),
		}})
	}

	if pages > 1 {
		var nav []sender.Button
		if page > 0 {
			nav = append(nav, sender.Button{Text: "◀", Data: fmt.Sprintf("memory:%d", page-1)})
		}
		nav = append(nav, sender.Button{Text: fmt.Sprintf("%d/%d", page+1, pages), Data: fmt.Sprintf("memory:%d", page)})
		if page < pages-1 {
			nav = append(nav, sender.Button{Text: "▶", Data: fmt.Sprintf("memory:%d", page+1)})
		}
		rows = append(rows, nav)
	}

	rows = append(rows, []sender.Button{memoryToggleButton(ch)})
	rows = append(rows, []sender.Button{{Text: "⬅ Меню", Data: "menu:"}})

	return []sender.Response{{
		Text:    fmt.Sprintf("🧠 Суперпамять: %s\nУзлов: %d (в индексе: %d)\n\n↓ — есть вложенные узлы, 📌 — закреплён.", status, len(ch.MemoryNodes), len(roots)),
		Buttons: rows,
	}}
}

// memoryNodeView renders one node: summary, children and controls.
func memoryNodeView(ch *chat.Chat, id int) []sender.Response {
	n := ch.FindMemoryNode(id)
	if n == nil {
		return memoryIndexView(ch, 0)
	}

	var sb strings.Builder
	pin := ""
	if n.Pinned {
		pin = " 📌"
	}
	sb.WriteString(fmt.Sprintf("🧠 #%d — %s%s\n%s\n\n", n.ID, n.Hook, pin, n.Created.In(ch.Location()).Format("02.01.2006")))
	sb.WriteString(truncate(n.Summary, memoryHubTextCap))

	rows := make([][]sender.Button, 0, len(n.Children)+3)
	for _, cid := range n.Children {
		if child := ch.FindMemoryNode(cid); child != nil {
			rows = append(rows, []sender.Button{{
				Text: "↓ " + truncate(child.Hook, 40),
				Data: fmt.Sprintf("memory:node:%d", child.ID),
			}})
		}
	}

	var controls []sender.Button
	if n.To > n.From {
		controls = append(controls, sender.Button{Text: "📜 Источник", Data: fmt.Sprintf("memory:src:%d", n.ID)})
	}
	pinLabel := "📌 Закрепить"
	if n.Pinned {
		pinLabel = "📌 Открепить"
	}
	controls = append(controls, sender.Button{Text: pinLabel, Data: fmt.Sprintf("memory:pin:%d", n.ID)})
	controls = append(controls, sender.Button{Text: "🗑 Удалить", Data: fmt.Sprintf("memory:del:%d", n.ID)})
	rows = append(rows, controls)
	rows = append(rows, []sender.Button{{Text: "⬅ Назад", Data: "memory:"}})

	return []sender.Response{{Text: sb.String(), Buttons: rows}}
}

// memoryNodeDeleteConfirm asks for confirmation before deleting a node
// (skipped when the chat opted out of delete confirmations). Deleting removes
// the summary from memory; children become roots, the raw archive stays.
func memoryNodeDeleteConfirm(ch *chat.Chat, id int) []sender.Response {
	n := ch.FindMemoryNode(id)
	if n == nil {
		return memoryIndexView(ch, 0)
	}
	if ch.Settings.SkipDeleteConfirm {
		ch.RemoveMemoryNode(id)
		return memoryIndexView(ch, 0)
	}
	note := ""
	if len(n.Children) > 0 {
		note = fmt.Sprintf("\nВложенные узлы (%d) не удалятся — вернутся в индекс.", len(n.Children))
	}
	return []sender.Response{{
		Text: fmt.Sprintf("Удалить узел #%d «%s»?%s", n.ID, truncate(n.Hook, 60), note),
		Buttons: [][]sender.Button{{
			{Text: "🗑 Удалить", Data: fmt.Sprintf("memory:del:yes:%d", n.ID)},
			{Text: "Отмена", Data: fmt.Sprintf("memory:node:%d", n.ID)},
		}},
	}}
}

// memorySourceView renders the raw archived transcript behind a node.
func (c *CommandMemory) memorySourceView(ch *chat.Chat, id int) []sender.Response {
	n := ch.FindMemoryNode(id)
	if n == nil {
		return memoryIndexView(ch, 0)
	}
	back := [][]sender.Button{{{Text: "⬅ Назад", Data: fmt.Sprintf("memory:node:%d", n.ID)}}}

	if n.To <= n.From {
		return []sender.Response{{Text: "У этого узла нет привязанного сырого текста.", Buttons: back}}
	}
	if c.Archive == nil {
		return []sender.Response{{Text: "Архив недоступен.", Buttons: back}}
	}
	msgs, err := c.Archive.ReadRange(ch.ChatID, n.From, n.To)
	if err != nil {
		return []sender.Response{{Text: "Не удалось прочитать архив.", Buttons: back}}
	}

	text := service.MemorySourceForTool(ch, msgs)
	return []sender.Response{{
		Text:    fmt.Sprintf("📜 Источник #%d (%d строк):\n\n%s", n.ID, n.To-n.From, truncate(text, memoryHubTextCap)),
		Buttons: back,
	}}
}
