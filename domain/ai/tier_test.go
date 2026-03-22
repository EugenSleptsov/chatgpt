package ai

import (
	"strings"
	"testing"
)

// --- FindTier ---

func TestFindTier_ByID(t *testing.T) {
	tier := FindTier("basic")
	if tier == nil {
		t.Fatal("FindTier(basic) returned nil")
	}
	if tier.ID != "basic" {
		t.Fatalf("ID = %q, want basic", tier.ID)
	}
}

func TestFindTier_ByLabel(t *testing.T) {
	tier := FindTier("ai-fast")
	if tier == nil {
		t.Fatal("FindTier(ai-fast) returned nil")
	}
	if tier.ID != "fast" {
		t.Fatalf("ID = %q, want fast", tier.ID)
	}
}

func TestFindTier_NotFound(t *testing.T) {
	if tier := FindTier("nonexistent"); tier != nil {
		t.Fatalf("expected nil, got %+v", tier)
	}
}

// --- DefaultTier ---

func TestDefaultTier(t *testing.T) {
	tier := DefaultTier()
	if tier.ID != DefaultTierID {
		t.Fatalf("DefaultTier().ID = %q, want %q", tier.ID, DefaultTierID)
	}
}

// --- TierList ---

func TestTierList_ContainsAllTiers(t *testing.T) {
	list := TierList()
	for _, tier := range Tiers {
		if !strings.Contains(list, tier.Label) {
			t.Errorf("TierList() missing label %q", tier.Label)
		}
		if !strings.Contains(list, tier.Desc) {
			t.Errorf("TierList() missing desc %q", tier.Desc)
		}
	}
}
