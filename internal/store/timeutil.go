package store

import "time"

// parseLocalTime parses an ISO 8601 / RFC3339 timestamp string and converts
// it to local time. Returns the zero time for empty or unparseable strings.
func parseLocalTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}

	// Try RFC3339 first (most common from Claude Code logs)
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Local()
	}

	// Try RFC3339Nano
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.Local()
	}

	// Try bare date (YYYY-MM-DD) — assume start of day in local time
	if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
		return t
	}

	// Try ISO 8601 without timezone (assume local)
	if t, err := time.ParseInLocation("2006-01-02T15:04:05", s, time.Local); err == nil {
		return t
	}

	return time.Time{}
}

// startOfDay returns midnight (00:00:00) in local time for the given time.
func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
