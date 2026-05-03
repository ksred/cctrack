package calculator

import "testing"

// Pricing assertions against https://platform.claude.com/docs/en/about-claude/pricing.
// Every case bills exactly 1M tokens of one kind so the expected cost equals the
// per-MTok rate and a failure pinpoints which rate moved.
func TestCalculatePricing(t *testing.T) {
	cases := []struct {
		label  string
		model  string
		usage  TokenUsage
		expect float64
	}{
		{"Opus 4.7 input", "claude-opus-4-7", TokenUsage{InputTokens: 1_000_000}, 5.00},
		{"Opus 4.7 output", "claude-opus-4-7", TokenUsage{OutputTokens: 1_000_000}, 25.00},
		{"Opus 4.6 input", "claude-opus-4-6", TokenUsage{InputTokens: 1_000_000}, 5.00},
		{"Opus 4.5 input", "claude-opus-4-5-20251101", TokenUsage{InputTokens: 1_000_000}, 5.00},
		{"Opus 4.1 input", "claude-opus-4-1-20250805", TokenUsage{InputTokens: 1_000_000}, 15.00},
		{"Opus 4 input", "claude-opus-4-20250514", TokenUsage{InputTokens: 1_000_000}, 15.00},
		{"Sonnet 4.6 input", "claude-sonnet-4-6", TokenUsage{InputTokens: 1_000_000}, 3.00},
		{"Sonnet 4.6 output", "claude-sonnet-4-6", TokenUsage{OutputTokens: 1_000_000}, 15.00},
		{"Sonnet 4.5 output", "claude-sonnet-4-5-20250929", TokenUsage{OutputTokens: 1_000_000}, 15.00},
		{"Sonnet 4 input", "claude-sonnet-4-20250514", TokenUsage{InputTokens: 1_000_000}, 3.00},
		{"Haiku 4.5 input", "claude-haiku-4-5-20251001", TokenUsage{InputTokens: 1_000_000}, 1.00},
		{"Haiku 4.5 output", "claude-haiku-4-5-20251001", TokenUsage{OutputTokens: 1_000_000}, 5.00},
		{"Haiku 3.5 input", "claude-haiku-3-5-20241022", TokenUsage{InputTokens: 1_000_000}, 0.80},
		{"Opus 4.7 5m cache write", "claude-opus-4-7", TokenUsage{CacheWrite5mTokens: 1_000_000}, 6.25},
		{"Opus 4.7 1h cache write", "claude-opus-4-7", TokenUsage{CacheWrite1hTokens: 1_000_000}, 10.00},
		{"Sonnet 4 cache read", "claude-sonnet-4", TokenUsage{CacheReadTokens: 1_000_000}, 0.30},
		// Unknown model falls back to Sonnet 4 pricing.
		{"Unknown → fallback", "MLGO_0000501", TokenUsage{InputTokens: 1_000_000}, 3.00},
	}

	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			got := Calculate(c.model, c.usage).TotalCost
			if got != c.expect {
				t.Errorf("model=%q usage=%+v: want $%.4f, got $%.4f", c.model, c.usage, c.expect, got)
			}
		})
	}
}
