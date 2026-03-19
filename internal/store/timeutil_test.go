package store

import (
	"testing"
	"time"
)

func TestParseLocalTime_RFC3339(t *testing.T) {
	result := parseLocalTime("2026-03-19T10:00:00+11:00")
	if result.IsZero() {
		t.Fatal("expected non-zero time for RFC3339 input")
	}
	// The parsed time should represent the same instant regardless of local zone
	expected := time.Date(2026, 3, 18, 23, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestParseLocalTime_RFC3339_UTC(t *testing.T) {
	result := parseLocalTime("2026-03-19T10:00:00Z")
	if result.IsZero() {
		t.Fatal("expected non-zero time for UTC input")
	}
	expected := time.Date(2026, 3, 19, 10, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestParseLocalTime_BareDate(t *testing.T) {
	result := parseLocalTime("2026-03-19")
	if result.IsZero() {
		t.Fatal("expected non-zero time for bare date")
	}
	// Bare dates should be interpreted as local midnight
	y, m, d := result.Date()
	if y != 2026 || m != 3 || d != 19 {
		t.Errorf("expected 2026-03-19, got %d-%02d-%02d", y, m, d)
	}
	if result.Hour() != 0 || result.Minute() != 0 {
		t.Errorf("expected midnight, got %02d:%02d", result.Hour(), result.Minute())
	}
}

func TestParseLocalTime_Empty(t *testing.T) {
	result := parseLocalTime("")
	if !result.IsZero() {
		t.Error("expected zero time for empty string")
	}
}

func TestParseLocalTime_Malformed(t *testing.T) {
	result := parseLocalTime("not-a-timestamp")
	if !result.IsZero() {
		t.Error("expected zero time for malformed string")
	}
}

func TestParseLocalTime_ISO8601NoTZ(t *testing.T) {
	result := parseLocalTime("2026-03-19T14:30:00")
	if result.IsZero() {
		t.Fatal("expected non-zero time for ISO 8601 without TZ")
	}
	if result.Hour() != 14 || result.Minute() != 30 {
		t.Errorf("expected 14:30, got %02d:%02d", result.Hour(), result.Minute())
	}
}

func TestStartOfDay(t *testing.T) {
	input := time.Date(2026, 3, 19, 14, 30, 45, 123, time.Local)
	result := startOfDay(input)
	expected := time.Date(2026, 3, 19, 0, 0, 0, 0, time.Local)
	if !result.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}
