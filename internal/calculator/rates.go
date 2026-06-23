package calculator

type ModelRates struct {
	Family              string
	InputPerMToken      float64
	OutputPerMToken     float64
	CacheReadPerMToken  float64
	CacheWritePerMToken float64
}

// Rates maps model family prefixes to their pricing (USD per million tokens).
// Models are matched by prefix: "claude-haiku-4-5-20251001" → "claude-haiku-4"
//
// Cache read is 0.1x input; cache write (ephemeral 5m) is 1.25x input — the
// standard Anthropic ratios. More specific families are listed first so prefix
// matching picks them before shorter prefixes.
//
// NOTE: "claude-opus-4" covers Opus 4.5/4.6/4.7/4.8 at $5/$25. The retired
// Opus 4.0/4.1 were $15/$75 — if you need those, add a dated entry above this one.
var Rates = []ModelRates{
	{
		Family:              "claude-fable-5",
		InputPerMToken:      10.00,
		OutputPerMToken:     50.00,
		CacheReadPerMToken:  1.00,
		CacheWritePerMToken: 12.50,
	},
	{
		Family:              "claude-mythos-5",
		InputPerMToken:      10.00,
		OutputPerMToken:     50.00,
		CacheReadPerMToken:  1.00,
		CacheWritePerMToken: 12.50,
	},
	{
		Family:              "claude-opus-4",
		InputPerMToken:      5.00,
		OutputPerMToken:     25.00,
		CacheReadPerMToken:  0.50,
		CacheWritePerMToken: 6.25,
	},
	{
		Family:              "claude-sonnet-4",
		InputPerMToken:      3.00,
		OutputPerMToken:     15.00,
		CacheReadPerMToken:  0.30,
		CacheWritePerMToken: 3.75,
	},
	{
		Family:              "claude-haiku-4",
		InputPerMToken:      1.00,
		OutputPerMToken:     5.00,
		CacheReadPerMToken:  0.10,
		CacheWritePerMToken: 1.25,
	},
}

func GetRates(model string) *ModelRates {
	for i := range Rates {
		if len(model) >= len(Rates[i].Family) && model[:len(Rates[i].Family)] == Rates[i].Family {
			return &Rates[i]
		}
	}
	// Fallback for unknown models: default to sonnet rates as the most common.
	for i := range Rates {
		if Rates[i].Family == "claude-sonnet-4" {
			return &Rates[i]
		}
	}
	return &Rates[0]
}
