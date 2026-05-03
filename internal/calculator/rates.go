package calculator

// RatesVersion / RatesUpdated identify the bundled rate card. Bump both whenever
// a rate changes or a model is added — the dashboard surfaces them so users can
// tell at a glance whether their build is on stale pricing.
const (
	RatesVersion = "v1.3"
	RatesUpdated = "2026-05-03"
)

type ModelRates struct {
	Family                string
	Released              string // YYYY-MM-DD; empty if unknown
	InputPerMToken        float64
	OutputPerMToken       float64
	CacheReadPerMToken    float64
	CacheWrite5mPerMToken float64
	CacheWrite1hPerMToken float64
}

// Rates is matched top-to-bottom by prefix, so list more-specific families
// first (e.g. "claude-opus-4-7" before "claude-opus-4-1" before "claude-opus-4").
// Source: https://platform.claude.com/docs/en/about-claude/pricing
var Rates = []ModelRates{
	// Opus 4.5 / 4.6 / 4.7 — current tier (3x cheaper than the original Opus 4 / 4.1).
	{Family: "claude-opus-4-7", Released: "2026-04-16", InputPerMToken: 5.00, OutputPerMToken: 25.00, CacheReadPerMToken: 0.50, CacheWrite5mPerMToken: 6.25, CacheWrite1hPerMToken: 10.00},
	{Family: "claude-opus-4-6", Released: "2026-02-05", InputPerMToken: 5.00, OutputPerMToken: 25.00, CacheReadPerMToken: 0.50, CacheWrite5mPerMToken: 6.25, CacheWrite1hPerMToken: 10.00},
	{Family: "claude-opus-4-5", Released: "2025-11-24", InputPerMToken: 5.00, OutputPerMToken: 25.00, CacheReadPerMToken: 0.50, CacheWrite5mPerMToken: 6.25, CacheWrite1hPerMToken: 10.00},
	// Opus 4 / 4.1 — original tier.
	{Family: "claude-opus-4-1", Released: "2025-08-05", InputPerMToken: 15.00, OutputPerMToken: 75.00, CacheReadPerMToken: 1.50, CacheWrite5mPerMToken: 18.75, CacheWrite1hPerMToken: 30.00},
	{Family: "claude-opus-4", Released: "2025-05-22", InputPerMToken: 15.00, OutputPerMToken: 75.00, CacheReadPerMToken: 1.50, CacheWrite5mPerMToken: 18.75, CacheWrite1hPerMToken: 30.00},
	// Sonnet 4 / 4.5 / 4.6 — same pricing across the family today, but listed
	// explicitly so a future per-version repricing is a one-line change rather
	// than a silent misattribution under the blanket prefix.
	{Family: "claude-sonnet-4-6", Released: "2026-02-17", InputPerMToken: 3.00, OutputPerMToken: 15.00, CacheReadPerMToken: 0.30, CacheWrite5mPerMToken: 3.75, CacheWrite1hPerMToken: 6.00},
	{Family: "claude-sonnet-4-5", Released: "2025-09-29", InputPerMToken: 3.00, OutputPerMToken: 15.00, CacheReadPerMToken: 0.30, CacheWrite5mPerMToken: 3.75, CacheWrite1hPerMToken: 6.00},
	{Family: "claude-sonnet-4", Released: "2025-05-22", InputPerMToken: 3.00, OutputPerMToken: 15.00, CacheReadPerMToken: 0.30, CacheWrite5mPerMToken: 3.75, CacheWrite1hPerMToken: 6.00},
	// Haiku 4.5.
	{Family: "claude-haiku-4-5", Released: "2025-10-15", InputPerMToken: 1.00, OutputPerMToken: 5.00, CacheReadPerMToken: 0.10, CacheWrite5mPerMToken: 1.25, CacheWrite1hPerMToken: 2.00},
	// Haiku 3.5 (still listed by Anthropic).
	{Family: "claude-haiku-3-5", Released: "2024-10-22", InputPerMToken: 0.80, OutputPerMToken: 4.00, CacheReadPerMToken: 0.08, CacheWrite5mPerMToken: 1.00, CacheWrite1hPerMToken: 1.60},
	// Deprecated — may appear in older logs.
	{Family: "claude-sonnet-3-7", Released: "2025-02-24", InputPerMToken: 3.00, OutputPerMToken: 15.00, CacheReadPerMToken: 0.30, CacheWrite5mPerMToken: 3.75, CacheWrite1hPerMToken: 6.00},
	{Family: "claude-opus-3", Released: "2024-03-04", InputPerMToken: 15.00, OutputPerMToken: 75.00, CacheReadPerMToken: 1.50, CacheWrite5mPerMToken: 18.75, CacheWrite1hPerMToken: 30.00},
	{Family: "claude-haiku-3", Released: "2024-03-13", InputPerMToken: 0.25, OutputPerMToken: 1.25, CacheReadPerMToken: 0.03, CacheWrite5mPerMToken: 0.30, CacheWrite1hPerMToken: 0.50},
}

// sonnetFallback is used when no Family prefix matches; chosen as the most
// common Claude model so the error mode is "small over-estimate" rather than zero.
var sonnetFallback = ModelRates{
	Family: "claude-sonnet-4 (fallback)", InputPerMToken: 3.00, OutputPerMToken: 15.00,
	CacheReadPerMToken: 0.30, CacheWrite5mPerMToken: 3.75, CacheWrite1hPerMToken: 6.00,
}

func GetRates(model string) *ModelRates {
	for i := range Rates {
		f := Rates[i].Family
		if len(model) >= len(f) && model[:len(f)] == f {
			return &Rates[i]
		}
	}
	return &sonnetFallback
}
