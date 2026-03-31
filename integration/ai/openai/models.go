package openai

import "GPTBot/domain/ai"

// modelSpec holds the concrete OpenAI API model name, per-token pricing,
// and context window size (used for auto-compact threshold).
type modelSpec struct {
	APIModel      string
	PriceIn       float64 // USD per 1 M input tokens
	PriceOut      float64 // USD per 1 M output tokens
	ContextWindow int     // max input tokens (used by auto-compact)
}

// models maps abstract tier IDs to concrete OpenAI specs.
// To upgrade models or pricing — change the values here, nothing else.
var models = map[string]modelSpec{
	"basic":   {APIModel: "gpt-5.4-nano", PriceIn: 0.20, PriceOut: 1.25, ContextWindow: 128_000},
	"fast":    {APIModel: "gpt-5.4-mini", PriceIn: 0.75, PriceOut: 4.50, ContextWindow: 64_000},
	"premium": {APIModel: "gpt-5.4", PriceIn: 2.50, PriceOut: 15.00, ContextWindow: 64_000},
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
