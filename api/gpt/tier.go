package gpt

// Tier represents a user-facing model option in /model command.
// Users pick a stable tier name; the underlying API model can be swapped at any time.
type Tier struct {
	ID       string // stored in chat settings (e.g. "basic")
	Label    string // display name shown to users (e.g. "gpt-basic")
	Desc     string // human-readable description
	APIModel string // actual OpenAI API model name (e.g. "gpt-5.4-nano")
}

// Tiers lists available user-facing tiers.
// To upgrade models — change APIModel here, nothing else.
var Tiers = []Tier{
	{ID: "basic", Label: "gpt-basic", Desc: "Экономичная, для простых задач", APIModel: "gpt-5.4-nano"},
	{ID: "fast", Label: "gpt-fast", Desc: "Быстрая, для кодинга и агентов", APIModel: "gpt-5.4-mini"},
	{ID: "premium", Label: "gpt-premium", Desc: "Максимальное качество рассуждений", APIModel: "gpt-5.4"},
}

const (
	DefaultTierID      = "basic"   // default tier for new chats
	VisionTierID       = "premium" // tier used for image analysis
	ImageEnhanceTierID = "basic"   // tier used for image prompt enhancement

	DefaultTemperature = 0.8 // default temperature for GPT calls
)

// legacyTierMap maps old model IDs (stored in existing chats) to current tier IDs.
var legacyTierMap = map[string]string{
	// GPT-3.x
	"gpt-3": "basic", "gpt-3.5-turbo": "basic", "gpt-3.5-turbo-1106": "basic",
	"gpt-3.5-turbo-16k": "basic", "gpt-316": "basic",
	// GPT-4.x
	"gpt-4": "basic", "gpt-4-turbo": "fast", "gpt-4-turbo-preview": "fast",
	"gpt-4-vision-preview": "premium", "gpt-4o": "fast", "gpt-4o-mini": "basic",
	// GPT-4.1
	"4.1-nano": "basic", "4.1-mini": "fast", "4.1": "premium",
	// GPT-5.x
	"gpt-5": "premium", "gpt-5-mini": "fast", "gpt-5-nano": "basic",
	"5.4-nano": "basic", "5.4-mini": "fast", "5.4": "premium", "5.4-pro": "premium",
	// o-series
	"o3-mini": "fast", "o3": "premium", "o4-mini": "fast",
}

// FindTier looks up a tier by ID or Label (handles legacy model IDs).
// Returns nil if not found.
func FindTier(id string) *Tier {
	if newID, ok := legacyTierMap[id]; ok {
		id = newID
	}
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

// ResolveAPIName maps any tier/model ID (including legacy) to the actual OpenAI API name.
func ResolveAPIName(id string) string {
	if t := FindTier(id); t != nil {
		return t.APIModel
	}
	return DefaultTier().APIModel
}

// TierList returns a formatted string listing all available tiers.
func TierList() string {
	var result string
	for _, t := range Tiers {
		result += t.Label + " — " + t.Desc + "\n"
	}
	return result
}
