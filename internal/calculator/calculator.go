package calculator

type TokenUsage struct {
	InputTokens        int64
	OutputTokens       int64
	CacheReadTokens    int64
	CacheWrite5mTokens int64
	CacheWrite1hTokens int64
}

type CostBreakdown struct {
	InputCost      float64
	OutputCost     float64
	CacheReadCost  float64
	CacheWriteCost float64 // sum of 5m + 1h cache-write cost
	TotalCost      float64
}

func Calculate(model string, usage TokenUsage) CostBreakdown {
	rates := GetRates(model)
	cw5m := float64(usage.CacheWrite5mTokens) / 1_000_000 * rates.CacheWrite5mPerMToken
	cw1h := float64(usage.CacheWrite1hTokens) / 1_000_000 * rates.CacheWrite1hPerMToken
	cb := CostBreakdown{
		InputCost:      float64(usage.InputTokens) / 1_000_000 * rates.InputPerMToken,
		OutputCost:     float64(usage.OutputTokens) / 1_000_000 * rates.OutputPerMToken,
		CacheReadCost:  float64(usage.CacheReadTokens) / 1_000_000 * rates.CacheReadPerMToken,
		CacheWriteCost: cw5m + cw1h,
	}
	cb.TotalCost = cb.InputCost + cb.OutputCost + cb.CacheReadCost + cb.CacheWriteCost
	return cb
}

func (u TokenUsage) Total() int64 {
	return u.InputTokens + u.OutputTokens + u.CacheReadTokens + u.CacheWrite5mTokens + u.CacheWrite1hTokens
}
