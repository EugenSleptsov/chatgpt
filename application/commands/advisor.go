package commands

import (
	"GPTBot/domain/chat"
	"GPTBot/pipeline"
	"GPTBot/pipeline/sender"
	"fmt"
	"strconv"
	"strings"
)

// CommandAdvisor is the button hub for advisor notes — auto-captured notes the
// assistant files under model-named topics (via the save_note tool). The hub
// lets the user browse topics, read entries, delete individual entries or
// whole topics, and merge two topics.
//
// Sub-routes (callback data "advisor:<sub>"):
//
//	""                      → topic list, first page
//	"<n>"                   → topic list, page n
//	"topic:<id>"            → entries of topic <id>
//	"del:<id>"              → per-entry delete picker for topic <id>
//	"rm:<id>:<eid>"         → delete entry <eid>, re-render picker
//	"deltopic:<id>"         → confirm topic deletion (honours SkipDeleteConfirm)
//	"deltopic:yes:<id>"     → delete topic, back to list
//	"merge"                 → pick source topic
//	"merge:<src>"           → pick destination topic
//	"merge:<src>:<dst>"     → merge source into destination, back to list
type CommandAdvisor struct{}

func (c *CommandAdvisor) Name() string { return "advisor" }
func (c *CommandAdvisor) Description() string {
	return "Заметки-советник (кнопки)."
}
func (c *CommandAdvisor) IsAdmin() bool { return false }

// advisorTopicsPerPage caps how many topic buttons fit on one keyboard page.
const advisorTopicsPerPage = 6

func (c *CommandAdvisor) Execute(ctx *pipeline.RequestContext, ch *chat.Chat) []sender.Response {
	args := strings.TrimSpace(ctx.CommandArgs)

	switch {
	case strings.HasPrefix(args, "topic:"):
		if id, err := strconv.Atoi(args[len("topic:"):]); err == nil {
			return advisorTopicView(ch, id)
		}
	case strings.HasPrefix(args, "del:"):
		if id, err := strconv.Atoi(args[len("del:"):]); err == nil {
			return advisorEntryDeleteView(ch, id)
		}
	case strings.HasPrefix(args, "rm:"):
		return advisorRemoveEntry(ch, args[len("rm:"):])
	case strings.HasPrefix(args, "deltopic:yes:"):
		if id, err := strconv.Atoi(args[len("deltopic:yes:"):]); err == nil {
			ch.RemoveAdvisorTopic(id)
		}
		return advisorTopicsView(ch, 0)
	case strings.HasPrefix(args, "deltopic:"):
		if id, err := strconv.Atoi(args[len("deltopic:"):]); err == nil {
			return advisorTopicDeleteConfirm(ch, id)
		}
	case args == "merge":
		return advisorMergePickView(ch, 0)
	case strings.HasPrefix(args, "merge:"):
		return advisorMerge(ch, args[len("merge:"):])
	case args != "":
		if page, err := strconv.Atoi(args); err == nil {
			return advisorTopicsView(ch, page)
		}
	}

	return advisorTopicsView(ch, 0)
}

// advisorTopicsView renders one page of the topic list.
func advisorTopicsView(ch *chat.Chat, page int) []sender.Response {
	total := len(ch.AdvisorTopics)
	if total == 0 {
		return []sender.Response{{
			Text: "📒 Заметок пока нет.\n\nПопросите бота что-нибудь записать («запиши…», «напомни мне про…») — он сам создаст тему и сохранит заметку.",
			Buttons: [][]sender.Button{
				{{Text: "⬅ Меню", Data: "menu:"}},
			},
		}}
	}

	pages := (total + advisorTopicsPerPage - 1) / advisorTopicsPerPage
	if page < 0 {
		page = 0
	}
	if page >= pages {
		page = pages - 1
	}
	start := page * advisorTopicsPerPage
	end := start + advisorTopicsPerPage
	if end > total {
		end = total
	}

	var sb strings.Builder
	sb.WriteString("📒 Заметки:\n\n")

	rows := make([][]sender.Button, 0, end-start+3)
	for _, t := range ch.AdvisorTopics[start:end] {
		sb.WriteString(fmt.Sprintf("• %s — %d зап.\n", t.Name, len(t.Entries)))
		rows = append(rows, []sender.Button{{
			Text: fmt.Sprintf("%s (%d)", t.Name, len(t.Entries)),
			Data: fmt.Sprintf("advisor:topic:%d", t.ID),
		}})
	}

	if total > 1 {
		rows = append(rows, []sender.Button{{Text: "🔀 Объединить темы", Data: "advisor:merge"}})
	}

	if pages > 1 {
		var nav []sender.Button
		if page > 0 {
			nav = append(nav, sender.Button{Text: "◀", Data: fmt.Sprintf("advisor:%d", page-1)})
		}
		nav = append(nav, sender.Button{Text: fmt.Sprintf("%d/%d", page+1, pages), Data: fmt.Sprintf("advisor:%d", page)})
		if page < pages-1 {
			nav = append(nav, sender.Button{Text: "▶", Data: fmt.Sprintf("advisor:%d", page+1)})
		}
		rows = append(rows, nav)
	}

	rows = append(rows, []sender.Button{{Text: "⬅ Меню", Data: "menu:"}})
	return []sender.Response{{Text: sb.String(), Buttons: rows}}
}

// advisorTopicView renders all entries of one topic.
func advisorTopicView(ch *chat.Chat, topicID int) []sender.Response {
	t := ch.FindAdvisorTopic(topicID)
	if t == nil {
		return advisorTopicsView(ch, 0)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📒 %s (%d):\n\n", t.Name, len(t.Entries)))
	for i, e := range t.Entries {
		sb.WriteString(fmt.Sprintf("%d. %s — %s\n", i+1, e.Created.Format("02.01"), e.Text))
	}

	return []sender.Response{{
		Text: sb.String(),
		Buttons: [][]sender.Button{
			{
				{Text: "🗑 Записи", Data: fmt.Sprintf("advisor:del:%d", t.ID)},
				{Text: "🗑 Тема", Data: fmt.Sprintf("advisor:deltopic:%d", t.ID)},
			},
			{{Text: "⬅ Назад", Data: "advisor:"}},
		},
	}}
}

// advisorEntryDeleteView renders one delete button per entry of the topic.
func advisorEntryDeleteView(ch *chat.Chat, topicID int) []sender.Response {
	t := ch.FindAdvisorTopic(topicID)
	if t == nil {
		return advisorTopicsView(ch, 0)
	}

	rows := make([][]sender.Button, 0, len(t.Entries)+1)
	for _, e := range t.Entries {
		rows = append(rows, []sender.Button{{
			Text: "🗑 " + truncate(e.Text, 32),
			Data: fmt.Sprintf("advisor:rm:%d:%d", t.ID, e.ID),
		}})
	}
	rows = append(rows, []sender.Button{{Text: "⬅ Назад", Data: fmt.Sprintf("advisor:topic:%d", t.ID)}})

	return []sender.Response{{
		Text:    fmt.Sprintf("Выберите запись для удаления (%s):", t.Name),
		Buttons: rows,
	}}
}

// advisorRemoveEntry handles "rm:<topicID>:<entryID>": deletes the entry and
// re-renders the delete picker (or the topic list when the topic is gone).
func advisorRemoveEntry(ch *chat.Chat, arg string) []sender.Response {
	tidStr, eidStr, ok := strings.Cut(arg, ":")
	if !ok {
		return advisorTopicsView(ch, 0)
	}
	tid, err1 := strconv.Atoi(tidStr)
	eid, err2 := strconv.Atoi(eidStr)
	if err1 != nil || err2 != nil {
		return advisorTopicsView(ch, 0)
	}
	ch.RemoveAdvisorEntry(tid, eid)
	if ch.FindAdvisorTopic(tid) == nil {
		return advisorTopicsView(ch, 0)
	}
	return advisorEntryDeleteView(ch, tid)
}

// advisorTopicDeleteConfirm asks for confirmation before dropping a whole
// topic (skipped when the chat opted out of delete confirmations).
func advisorTopicDeleteConfirm(ch *chat.Chat, topicID int) []sender.Response {
	t := ch.FindAdvisorTopic(topicID)
	if t == nil {
		return advisorTopicsView(ch, 0)
	}
	if ch.Settings.SkipDeleteConfirm {
		ch.RemoveAdvisorTopic(topicID)
		return advisorTopicsView(ch, 0)
	}
	return []sender.Response{{
		Text: fmt.Sprintf("Удалить тему «%s» со всеми записями (%d)?", t.Name, len(t.Entries)),
		Buttons: [][]sender.Button{{
			{Text: "🗑 Удалить", Data: fmt.Sprintf("advisor:deltopic:yes:%d", t.ID)},
			{Text: "Отмена", Data: fmt.Sprintf("advisor:topic:%d", t.ID)},
		}},
	}}
}

// advisorMergePickView renders the source-topic picker for a merge.
func advisorMergePickView(ch *chat.Chat, _ int) []sender.Response {
	rows := make([][]sender.Button, 0, len(ch.AdvisorTopics)+1)
	for _, t := range ch.AdvisorTopics {
		rows = append(rows, []sender.Button{{
			Text: fmt.Sprintf("%s (%d)", t.Name, len(t.Entries)),
			Data: fmt.Sprintf("advisor:merge:%d", t.ID),
		}})
	}
	rows = append(rows, []sender.Button{{Text: "⬅ Назад", Data: "advisor:"}})
	return []sender.Response{{
		Text:    "🔀 Какую тему влить в другую? (она будет удалена, записи переедут)",
		Buttons: rows,
	}}
}

// advisorMerge handles "merge:<src>" (pick destination) and
// "merge:<src>:<dst>" (perform the merge).
func advisorMerge(ch *chat.Chat, arg string) []sender.Response {
	srcStr, dstStr, hasDst := strings.Cut(arg, ":")
	src, err := strconv.Atoi(srcStr)
	if err != nil {
		return advisorTopicsView(ch, 0)
	}

	if hasDst {
		if dst, err := strconv.Atoi(dstStr); err == nil {
			ch.MergeAdvisorTopics(src, dst)
		}
		return advisorTopicsView(ch, 0)
	}

	srcTopic := ch.FindAdvisorTopic(src)
	if srcTopic == nil {
		return advisorTopicsView(ch, 0)
	}
	rows := make([][]sender.Button, 0, len(ch.AdvisorTopics))
	for _, t := range ch.AdvisorTopics {
		if t.ID == src {
			continue
		}
		rows = append(rows, []sender.Button{{
			Text: fmt.Sprintf("→ %s (%d)", t.Name, len(t.Entries)),
			Data: fmt.Sprintf("advisor:merge:%d:%d", src, t.ID),
		}})
	}
	rows = append(rows, []sender.Button{{Text: "⬅ Назад", Data: "advisor:merge"}})
	return []sender.Response{{
		Text:    fmt.Sprintf("Куда перенести записи из «%s»?", srcTopic.Name),
		Buttons: rows,
	}}
}

// truncate shortens s to max runes, appending an ellipsis when cut.
func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}
