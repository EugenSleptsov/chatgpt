package service

import (
	chatdomain "GPTBot/domain/chat"
	"fmt"
	"log"
	"strings"
	"time"
)

// Supermemory service layer: archiving the raw transcript, rendering the
// system-prompt index and answering the drill-down tools. The whole subsystem
// is gated by ChatSettings.Supermemory — every entry point below no-ops when
// the setting is off, so disabling hides memory without deleting it.

const (
	// supermemoryIndexLimit caps how many root hooks go into the system prompt.
	// When roots exceed this, the oldest fall off the index until
	// meta-compaction folds them into parent nodes (search still finds them).
	supermemoryIndexLimit = 40

	// supermemorySearchLimit caps search_memory results.
	supermemorySearchLimit = 10

	// supermemorySourceCap caps the characters returned by read_memory_source
	// so a huge raw range cannot blow up the tool output.
	supermemorySourceCap = 4000
)

// SupermemoryPrompt renders the L0 index for the system prompt: one hook line
// per root node. Empty when the feature is off or there are no nodes yet.
func SupermemoryPrompt(chat *chatdomain.Chat) string {
	if !chat.Settings.Supermemory {
		return ""
	}
	roots := chat.RootMemoryNodes()
	if len(roots) == 0 {
		return ""
	}
	if len(roots) > supermemoryIndexLimit {
		roots = roots[len(roots)-supermemoryIndexLimit:]
	}
	var sb strings.Builder
	sb.WriteString("Supermemory index — long-term memory of past conversations, one hook per node. Drill down with search_memory (keyword), read_memory (summary by node id) and read_memory_source (raw transcript):\n")
	for _, n := range roots {
		sb.WriteString(fmt.Sprintf("#%d — %s\n", n.ID, n.Hook))
	}
	return strings.TrimRight(sb.String(), "\n")
}

// ArchiveHistory appends every not-yet-archived line of the session history to
// the raw archive and stamps the entries with their [From, To) ranges. Safe to
// call repeatedly — archived entries are skipped. Also picks up a response
// that was attached after its prompt line had already been archived.
// Failures are logged and left for the next call to retry; archiving must
// never break the chat flow.
func ArchiveHistory(archive chatdomain.Archive, chat *chatdomain.Chat, session *chatdomain.Session) {
	if archive == nil || !chat.Settings.Supermemory {
		return
	}
	for _, e := range session.History {
		switch {
		case e.ArchiveTo == 0 && e.Prompt.Content != "":
			// Entry not archived yet: prompt line (+response when present).
			msgs := []chatdomain.ArchivedMessage{archivedMessage(session.ID, e.Prompt)}
			if e.Response.Content != "" {
				msgs = append(msgs, archivedMessage(session.ID, e.Response))
			}
			from, err := archive.Append(chat.ChatID, msgs)
			if err != nil {
				log.Printf("[Supermemory] archive append failed: %v", err)
				return
			}
			e.ArchiveFrom, e.ArchiveTo = from, from+len(msgs)

		case e.ArchiveTo == e.ArchiveFrom+1 && e.Response.Content != "":
			// Prompt was archived earlier, response arrived after: append it.
			from, err := archive.Append(chat.ChatID, []chatdomain.ArchivedMessage{archivedMessage(session.ID, e.Response)})
			if err != nil {
				log.Printf("[Supermemory] archive append failed: %v", err)
				return
			}
			if from == e.ArchiveTo {
				e.ArchiveTo = from + 1
			} else {
				// Should not happen (per-chat jobs are serialized); keep the
				// range honest rather than spanning foreign lines.
				log.Printf("[Supermemory] non-contiguous response line %d for entry range [%d,%d)", from, e.ArchiveFrom, e.ArchiveTo)
			}
		}
	}
}

func archivedMessage(sessionID int, m chatdomain.Message) chatdomain.ArchivedMessage {
	role := m.Role
	if role == "" {
		role = "user"
	}
	return chatdomain.ArchivedMessage{
		Time:      time.Now(),
		SessionID: sessionID,
		Role:      role,
		Content:   m.Content,
	}
}

// SearchMemoryNodes returns up to supermemorySearchLimit nodes whose hook or
// summary contains the query (case-insensitive). Search covers all layers,
// including nodes that fell off the prompt index.
func SearchMemoryNodes(chat *chatdomain.Chat, query string) []*chatdomain.MemoryNode {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	var out []*chatdomain.MemoryNode
	for _, n := range chat.MemoryNodes {
		if strings.Contains(strings.ToLower(n.Hook), q) || strings.Contains(strings.ToLower(n.Summary), q) {
			out = append(out, n)
			if len(out) == supermemorySearchLimit {
				break
			}
		}
	}
	return out
}

// MemoryNodeForTool renders one node for the read_memory tool output:
// summary body, child hooks (descend further) and a raw-source pointer.
func MemoryNodeForTool(chat *chatdomain.Chat, n *chatdomain.MemoryNode) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Node #%d — %s\nCreated: %s\n\n%s\n", n.ID, n.Hook, n.Created.In(chat.Location()).Format("2006-01-02"), n.Summary))
	if len(n.Children) > 0 {
		sb.WriteString("\nChildren (read_memory to descend):\n")
		for _, id := range n.Children {
			if c := chat.FindMemoryNode(id); c != nil {
				sb.WriteString(fmt.Sprintf("#%d — %s\n", c.ID, c.Hook))
			}
		}
	}
	if n.To > n.From {
		sb.WriteString(fmt.Sprintf("\nRaw transcript available: call read_memory_source with node_id %d (%d lines).\n", n.ID, n.To-n.From))
	}
	return sb.String()
}

// MemorySourceForTool renders the raw archived transcript behind a node,
// capped at supermemorySourceCap characters.
func MemorySourceForTool(chat *chatdomain.Chat, msgs []chatdomain.ArchivedMessage) string {
	loc := chat.Location()
	var sb strings.Builder
	for _, m := range msgs {
		line := fmt.Sprintf("[%s] %s: %s\n", m.Time.In(loc).Format("2006-01-02 15:04"), m.Role, m.Content)
		if sb.Len()+len(line) > supermemorySourceCap {
			sb.WriteString(fmt.Sprintf("… (truncated, %d chars cap)\n", supermemorySourceCap))
			break
		}
		sb.WriteString(line)
	}
	return sb.String()
}

// splitHookSummary splits a compaction reply into the one-line hook requested
// by the compact prompt and the summary body. Falls back to a truncated first
// line when the model ignored the format.
func splitHookSummary(text string) (hook, summary string) {
	first, rest, _ := strings.Cut(strings.TrimSpace(text), "\n")
	hook = truncateRunes(strings.TrimSpace(first), 100)
	summary = strings.TrimSpace(rest)
	if summary == "" {
		summary = text
	}
	return hook, summary
}

// archiveRangeOfEntries returns the combined [From, To) archive range of the
// given entries. Zero range when none of them were archived.
func archiveRangeOfEntries(entries []*chatdomain.ConversationEntry) (from, to int) {
	first := true
	for _, e := range entries {
		if e.ArchiveTo == 0 {
			continue
		}
		if first || e.ArchiveFrom < from {
			from = e.ArchiveFrom
		}
		if e.ArchiveTo > to {
			to = e.ArchiveTo
		}
		first = false
	}
	if first {
		return 0, 0
	}
	return from, to
}
