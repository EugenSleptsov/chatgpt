package openai

import "testing"

func TestResolveModel_KnownTier(t *testing.T) {
	got := ResolveModel("premium")
	want := "gpt-5.4"
	if got != want {
		t.Fatalf("ResolveModel(premium) = %q, want %q", got, want)
	}
}

func TestResolveModel_Basic(t *testing.T) {
	got := ResolveModel("basic")
	want := "gpt-5.4-nano"
	if got != want {
		t.Fatalf("ResolveModel(basic) = %q, want %q", got, want)
	}
}

func TestResolveModel_Fast(t *testing.T) {
	got := ResolveModel("fast")
	want := "gpt-5.4-mini"
	if got != want {
		t.Fatalf("ResolveModel(fast) = %q, want %q", got, want)
	}
}

func TestResolveModel_ByLabel(t *testing.T) {
	got := ResolveModel("ai-premium")
	want := "gpt-5.4"
	if got != want {
		t.Fatalf("ResolveModel(ai-premium) = %q, want %q", got, want)
	}
}

func TestResolveModel_UnknownFallsToDefault(t *testing.T) {
	got := ResolveModel("nonexistent")
	want := "gpt-5.4-nano" // default tier = basic
	if got != want {
		t.Fatalf("ResolveModel(nonexistent) = %q, want default %q", got, want)
	}
}

// --- CostForTokens ---

func TestCostForTokens_Basic(t *testing.T) {
	cost := CostForTokens("basic", 1_000_000, 1_000_000)
	expected := 0.20 + 1.25
	if diff := cost - expected; diff > 0.0001 || diff < -0.0001 {
		t.Fatalf("CostForTokens(basic, 1M, 1M) = %f, want %f", cost, expected)
	}
}

func TestCostForTokens_Premium(t *testing.T) {
	cost := CostForTokens("premium", 500_000, 200_000)
	expected := 0.5*2.50 + 0.2*15.00
	if diff := cost - expected; diff > 0.0001 || diff < -0.0001 {
		t.Fatalf("CostForTokens(premium, 500K, 200K) = %f, want %f", cost, expected)
	}
}

func TestCostForTokens_UnknownTier(t *testing.T) {
	cost := CostForTokens("nonexistent", 100, 100)
	if cost != 0 {
		t.Fatalf("expected 0 for unknown tier, got %f", cost)
	}
}

func TestCostForTokens_ZeroTokens(t *testing.T) {
	cost := CostForTokens("basic", 0, 0)
	if cost != 0 {
		t.Fatalf("expected 0, got %f", cost)
	}
}
