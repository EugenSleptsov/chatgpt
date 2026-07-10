package chat

import "time"

// Supermemory: layered lossless long-term memory over the chat's raw
// transcript, LSM-tree style. Every message is archived verbatim (layer -1);
// auto-compaction turns evicted history chunks into MemoryNodes — a summary
// plus a pointer to the underlying raw range (layer 0); later meta-compaction
// merges nodes into parent nodes ("summary of summaries") without ever
// rewriting the children. Reads descend from the index via tools.
//
// The whole subsystem is gated by ChatSettings.Supermemory: when off, nothing
// is archived, the index is not loaded into the prompt and the tools are not
// exposed — existing nodes and archive files are kept untouched.

// ArchivedMessage is one raw transcript line in the supermemory archive.
type ArchivedMessage struct {
	Time      time.Time
	SessionID int
	Role      string
	Content   string
}

// Archive is the append-only raw-transcript store backing supermemory.
// Lines are addressed by their zero-based index; a range is [From, To).
// Implementations live in infrastructure/storage.
type Archive interface {
	// Append stores messages and returns the index of the first stored line.
	Append(chatID int64, msgs []ArchivedMessage) (from int, err error)
	// ReadRange returns the lines [from, to).
	ReadRange(chatID int64, from, to int) ([]ArchivedMessage, error)
	// DeleteRange is the reserved hook for future privacy tooling ("hard
	// forget"): implementations tombstone the lines in place. Not wired to any
	// user-facing command yet.
	DeleteRange(chatID int64, from, to int) error
}

// MemoryNode is one unit of supermemory: a summary with a one-line hook for
// the index and a link to its underlying source — either a raw archive range
// (leaf, produced by history compaction) or child nodes (produced by
// meta-compaction of the index).
type MemoryNode struct {
	ID        int
	Hook      string // one-line hook shown in the system-prompt index
	Summary   string // full summary body
	Children  []int  `json:",omitempty"` // deeper nodes compacted into this one
	From      int    `json:",omitempty"` // archive range [From, To) of the raw source
	To        int    `json:",omitempty"` // To == From means no raw source
	SessionID int    `json:",omitempty"` // session the source conversation happened in
	Created   time.Time
	Pinned    bool `json:",omitempty"` // protected from meta-compaction
}

// AddMemoryNode appends a new supermemory node and returns it.
func (c *Chat) AddMemoryNode(hook, summary string, from, to, sessionID int) *MemoryNode {
	if c.NextMemoryNodeID == 0 {
		c.NextMemoryNodeID = 1
	}
	n := &MemoryNode{
		ID:        c.NextMemoryNodeID,
		Hook:      hook,
		Summary:   summary,
		From:      from,
		To:        to,
		SessionID: sessionID,
		Created:   time.Now(),
	}
	c.NextMemoryNodeID++
	c.MemoryNodes = append(c.MemoryNodes, n)
	return n
}

// FindMemoryNode looks up a node by ID. Returns nil if not found.
func (c *Chat) FindMemoryNode(id int) *MemoryNode {
	for _, n := range c.MemoryNodes {
		if n.ID == id {
			return n
		}
	}
	return nil
}

// RemoveMemoryNode deletes one node and scrubs its ID from every other
// node's Children list (its own children thereby become roots again). The raw
// archive is untouched — this forgets the summary, not the transcript.
// Returns false if the node is not found.
func (c *Chat) RemoveMemoryNode(id int) bool {
	idx := -1
	for i, n := range c.MemoryNodes {
		if n.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return false
	}
	c.MemoryNodes = append(c.MemoryNodes[:idx], c.MemoryNodes[idx+1:]...)
	for _, n := range c.MemoryNodes {
		for i, cid := range n.Children {
			if cid == id {
				n.Children = append(n.Children[:i], n.Children[i+1:]...)
				break
			}
		}
	}
	return true
}

// RootMemoryNodes returns the nodes not referenced as any node's child —
// the top layer that forms the system-prompt index.
func (c *Chat) RootMemoryNodes() []*MemoryNode {
	child := make(map[int]bool)
	for _, n := range c.MemoryNodes {
		for _, id := range n.Children {
			child[id] = true
		}
	}
	roots := make([]*MemoryNode, 0, len(c.MemoryNodes))
	for _, n := range c.MemoryNodes {
		if !child[n.ID] {
			roots = append(roots, n)
		}
	}
	return roots
}
