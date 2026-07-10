package chat

import "testing"

func TestRemoveMemoryNode(t *testing.T) {
	c := &Chat{}
	a := c.AddMemoryNode("a", "…", 0, 0, 1)
	b := c.AddMemoryNode("b", "…", 0, 0, 1)
	parent := c.AddMemoryNode("parent", "…", 0, 0, 1)
	parent.Children = []int{a.ID, b.ID}

	// Removing a child scrubs it from the parent's Children.
	if !c.RemoveMemoryNode(a.ID) {
		t.Fatal("removal must succeed")
	}
	if len(parent.Children) != 1 || parent.Children[0] != b.ID {
		t.Fatalf("dangling child reference: %v", parent.Children)
	}

	// Removing the parent promotes the remaining child to a root.
	if !c.RemoveMemoryNode(parent.ID) {
		t.Fatal("removal must succeed")
	}
	roots := c.RootMemoryNodes()
	if len(roots) != 1 || roots[0].ID != b.ID {
		t.Fatalf("child must become a root, got %+v", roots)
	}

	if c.RemoveMemoryNode(999) {
		t.Fatal("removing a missing node must return false")
	}
}
