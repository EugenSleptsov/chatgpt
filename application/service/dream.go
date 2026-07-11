package service

import (
	"GPTBot/domain/ai"
	chatdomain "GPTBot/domain/chat"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// Dream — manual, model-driven reorganization of the supermemory root index
// (/dream). Unlike MetaCompact's deterministic oldest-first folding, dreaming
// lets the model pick thematic clusters: scattered fragments of one topic,
// duplicated information, index clutter. Grouping is lossless — child nodes
// are linked under a new parent, never rewritten or deleted.

const (
	// dreamMinRoots is the minimum number of non-pinned root nodes before a
	// dream is worth a GPT call.
	dreamMinRoots = 5

	// dreamSummaryCap bounds each node summary in the planning input so a big
	// index still fits in one call.
	dreamSummaryCap = 600
)

const dreamPrompt = `You are reorganizing the root index of a layered long-term memory.
Given root memory nodes (id, hook, summary), find groups worth folding into one parent node:
- several nodes covering the same or heavily overlapping topic (duplicated information),
- many scattered fragments belonging to one theme,
- related nodes that clutter the index and would read better as one entry.
Each group needs at least 2 node ids. A node may appear in at most one group. Never invent ids. Leave well-scoped nodes out. If the index is already fine, return an empty list.
For each group write a hook (one line, max 80 chars) and a merged summary that explicitly names every covered topic — the child nodes stay readable below the parent, so be a faithful table of contents rather than a replacement. Write hooks and summaries in the same language as the node summaries.
Respond with JSON only, no code fences, exactly this shape:
{"groups":[{"hook":"...","summary":"...","node_ids":[1,2]}]}`

// dreamGroup is one cluster in the model's reorganization plan.
type dreamGroup struct {
	Hook    string `json:"hook"`
	Summary string `json:"summary"`
	NodeIDs []int  `json:"node_ids"`
}

// dreamEligibleRoots returns the root nodes a dream may regroup: non-pinned
// roots (pinned nodes stay where the user fixed them).
func dreamEligibleRoots(chat *chatdomain.Chat) []*chatdomain.MemoryNode {
	var out []*chatdomain.MemoryNode
	for _, n := range chat.RootMemoryNodes() {
		if !n.Pinned {
			out = append(out, n)
		}
	}
	return out
}

// Dream asks the model for a regrouping plan over the eligible root nodes and
// applies the valid part of it. Returns the created parent nodes. No-op
// (nil, nil, nil) when the feature is off or there are too few roots to bother.
func (cs *CompactService) Dream(chat *chatdomain.Chat, model string) ([]*chatdomain.MemoryNode, *TokenUsage, error) {
	if chat == nil || !chat.Settings.Supermemory {
		return nil, nil, nil
	}
	eligible := dreamEligibleRoots(chat)
	if len(eligible) < dreamMinRoots {
		return nil, nil, nil
	}

	var sb strings.Builder
	sb.WriteString("Root memory nodes:\n\n")
	for _, n := range eligible {
		sb.WriteString(fmt.Sprintf("#%d — %s\n%s\n\n", n.ID, n.Hook, truncateRunes(n.Summary, dreamSummaryCap)))
	}

	resp, err := cs.GptClient.CallGPT([]ai.Message{{Role: "user", Content: sb.String()}}, model, dreamPrompt)
	if err != nil {
		log.Printf("[Dream] GPT error: %v", err)
		return nil, nil, err
	}
	var usage TokenUsage
	usage.add(extractUsage(resp, model, "Dream", cs.CostFn))

	groups, err := parseDreamPlan(resp.OutputText())
	if err != nil {
		log.Printf("[Dream] bad plan: %v", err)
		return nil, &usage, err
	}

	byID := make(map[int]*chatdomain.MemoryNode, len(eligible))
	for _, n := range eligible {
		byID[n.ID] = n
	}
	used := make(map[int]bool)
	var created []*chatdomain.MemoryNode
	for _, g := range groups {
		ids := make([]int, 0, len(g.NodeIDs))
		for _, id := range g.NodeIDs {
			if byID[id] != nil && !used[id] {
				ids = append(ids, id)
			}
		}
		if len(ids) < 2 || strings.TrimSpace(g.Hook) == "" {
			continue
		}
		for _, id := range ids {
			used[id] = true
		}
		summary := strings.TrimSpace(g.Summary)
		if summary == "" {
			summary = strings.TrimSpace(g.Hook)
		}
		parent := chat.AddMemoryNode(truncateRunes(strings.TrimSpace(g.Hook), 100), summary, 0, 0, 0)
		parent.Children = ids
		created = append(created, parent)
		log.Printf("[Dream] node #%d groups %d children: %q", parent.ID, len(ids), parent.Hook)
	}
	return created, &usage, nil
}

// parseDreamPlan extracts the JSON plan from a model reply, tolerating stray
// text or code fences around it.
func parseDreamPlan(text string) ([]dreamGroup, error) {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("no JSON object in reply")
	}
	var plan struct {
		Groups []dreamGroup `json:"groups"`
	}
	if err := json.Unmarshal([]byte(text[start:end+1]), &plan); err != nil {
		return nil, err
	}
	return plan.Groups, nil
}

// Dream runs a supermemory reorganization on the chat and renders a
// user-facing report; the second return is how many parent nodes were created
// (auto-dream stays silent when it is zero). Cost is accumulated on the chat
// like any model call. A successful run stamps LastDreamNodeID so the nightly
// auto-dream can tell whether anything new appeared since.
func (s *GPTService) Dream(chat *chatdomain.Chat) (string, int) {
	if len(dreamEligibleRoots(chat)) < dreamMinRoots {
		return fmt.Sprintf("Узлов пока мало (нужно хотя бы %d незакреплённых) — перераспределять нечего.", dreamMinRoots), 0
	}
	if s.Compact == nil {
		return "Реорганизация памяти недоступна.", 0
	}
	created, usage, err := s.Compact.Dream(chat, chat.ActiveSession().Model)
	if usage != nil {
		chat.AccumulateCost(usage.Cost, usage.InputTokens, usage.OutputTokens)
	}
	if err != nil {
		return "Сон не задался — не получилось перечитать память. Попробуйте позже.", 0
	}
	chat.LastDreamNodeID = chat.NextMemoryNodeID
	if len(created) == 0 {
		return "Память в порядке — сгруппировать нечего.", 0
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Память реорганизована, новых тем: %d\n", len(created)))
	for _, n := range created {
		sb.WriteString(fmt.Sprintf("• #%d %s — %d узл.\n", n.ID, n.Hook, len(n.Children)))
	}
	sb.WriteString("\nИсходные узлы не удалены — они вложены в темы (смотреть: /memory).")
	return sb.String(), len(created)
}
