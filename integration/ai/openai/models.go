package openai

import "GPTBot/domain/ai"

// modelSpec holds the concrete OpenAI API model name, per-token pricing,
// and context window size (used for auto-compact threshold).
type modelSpec struct {
	APIModel      string
	PriceIn       float64 // USD per 1 M input tokens
	PriceOut      float64 // USD per 1 M output tokens
	ContextWindow int     // max input tokens (used by auto-compact)
	Effort        string  // reasoning effort: "medium" | "high" (empty = omit reasoning)
}

// models maps abstract tier IDs to concrete OpenAI specs.
// To upgrade models or pricing — change the values here, nothing else.
var models = map[string]modelSpec{
	"basic":   {APIModel: "gpt-5.4-nano", PriceIn: 0.20, PriceOut: 1.25, ContextWindow: 128_000, Effort: "medium"},
	"fast":    {APIModel: "gpt-5.4-mini", PriceIn: 0.75, PriceOut: 4.50, ContextWindow: 400_000, Effort: "medium"},
	"premium": {APIModel: "gpt-5.5", PriceIn: 5.00, PriceOut: 30.00, ContextWindow: 1_000_000, Effort: "high"},
}

// ImageGenerationCost is the approximate per-image cost for DALL-E 3 1024×1024.
const ImageGenerationCost = 0.04

// ResolveModel maps a tier ID (or label) to the concrete OpenAI API model name.
// Unknown tiers fall back to the default tier's model.
func ResolveModel(tierID string) string {
	// Direct ID lookup.
	if m, ok := models[tierID]; ok {
		return m.APIModel
	}
	// Try resolving label → ID first.
	if t := ai.FindTier(tierID); t != nil {
		if m, ok := models[t.ID]; ok {
			return m.APIModel
		}
	}
	// Fallback to default tier.
	return models[ai.DefaultTierID].APIModel
}

// ReasoningForTier returns the reasoning config for a tier (ID or label), or nil
// when the tier has no effort set (non-reasoning model → field omitted).
func ReasoningForTier(tierID string) *ai.Reasoning {
	spec, ok := models[tierID]
	if !ok {
		if t := ai.FindTier(tierID); t != nil {
			spec, ok = models[t.ID]
		}
	}
	if !ok || spec.Effort == "" {
		return nil
	}
	return &ai.Reasoning{Effort: spec.Effort}
}

// CostForTokens calculates the USD cost for the given token counts on the specified tier.
func CostForTokens(tierID string, inputTokens, outputTokens int) float64 {
	spec, ok := models[tierID]
	if !ok {
		// Try resolving label → ID.
		if t := ai.FindTier(tierID); t != nil {
			spec, ok = models[t.ID]
		}
	}
	if !ok {
		return 0
	}
	return float64(inputTokens)/1_000_000*spec.PriceIn + float64(outputTokens)/1_000_000*spec.PriceOut
}

// ContextWindowForTier returns the context window size (max input tokens) for the given tier.
// Returns a conservative default (32K) for unknown tiers.
func ContextWindowForTier(tierID string) int {
	if m, ok := models[tierID]; ok {
		return m.ContextWindow
	}
	if t := ai.FindTier(tierID); t != nil {
		if m, ok := models[t.ID]; ok {
			return m.ContextWindow
		}
	}
	return 32_000 // safe default
}
