package ai

import "fmt"

// Tier represents a user-facing model option in /model command.
// Users pick a stable tier name; the underlying API model is resolved
// by the concrete provider implementation (e.g. integration/ai/openai).
type Tier struct {
	ID    string // stored in chat settings (e.g. "basic")
	Label string // display name shown to users (e.g. "ai-basic")
	Desc  string // human-readable description
}

// Tiers lists available user-facing tiers.
// Concrete API model names are resolved by the provider (see integration/ai/openai).
var Tiers = []Tier{
	{ID: "basic", Label: "ai-basic", Desc: "Экономичная, для простых задач"},
	{ID: "fast", Label: "ai-fast", Desc: "Быстрая, для кодинга и агентов"},
	{ID: "premium", Label: "ai-premium", Desc: "Максимальное качество рассуждений"},
}

const (
	DefaultTierID      = "basic"   // default tier for new chats
	VisionTierID       = "premium" // tier used for image analysis
	ImageEnhanceTierID = "basic"   // tier used for image prompt enhancement
)

// FindTier looks up a tier by ID or Label. Returns nil if not found.
func FindTier(id string) *Tier {
	for i := range Tiers {
		if Tiers[i].ID == id || Tiers[i].Label == id {
			return &Tiers[i]
		}
	}
	return nil
}

// DefaultTier returns the default tier.
func DefaultTier() Tier {
	return *FindTier(DefaultTierID)
}

// TierList returns a formatted string listing all available tiers.
func TierList() string {
	var result string
	for _, t := range Tiers {
		result += fmt.Sprintf("%s — %s\n", t.Label, t.Desc)
	}
	return result
}
