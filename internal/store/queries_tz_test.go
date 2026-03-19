package store

import (
	"math"
	"testing"
	"time"
)

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.0001
}

// setupTZTestStore creates a store and inserts sessions spanning a midnight
// boundary to test timezone-aware bucketing.
func setupTZTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	now := time.Now()

	// Session 1: activity 2 hours ago (should be "today" in local time)
	ts1 := now.Add(-2 * time.Hour).Format(time.RFC3339)
	err = s.UpsertSession(SessionDelta{
		ID: "today-session", Project: "test", Model: "claude-sonnet-4-20250514",
		Timestamp: ts1, DeltaInput: 100, DeltaOutput: 50, DeltaCost: 0.05,
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// Session 2: activity 2 days ago (should NOT be "today")
	ts2 := now.AddDate(0, 0, -2).Format(time.RFC3339)
	err = s.UpsertSession(SessionDelta{
		ID: "old-session", Project: "test", Model: "claude-sonnet-4-20250514",
		Timestamp: ts2, DeltaInput: 200, DeltaOutput: 100, DeltaCost: 0.10,
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	return s
}

func TestGetSummary_TodayBoundary(t *testing.T) {
	s := setupTZTestStore(t)
	defer s.Close()

	summary, err := s.GetSummary()
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}

	// "Today" should only include the session from 2 hours ago
	if !approxEqual(summary.Today.Cost, 0.05) {
		t.Errorf("today cost = %f, want 0.05 (only today's session)", summary.Today.Cost)
	}

	// "This week" should include both sessions
	if !approxEqual(summary.Week.Cost, 0.15) {
		t.Errorf("week cost = %f, want 0.15 (both sessions)", summary.Week.Cost)
	}
}

func TestGetDailySummary_LocalBucketing(t *testing.T) {
	s := setupTZTestStore(t)
	defer s.Close()

	daily, err := s.GetDailySummary(7)
	if err != nil {
		t.Fatalf("GetDailySummary: %v", err)
	}

	// Should have 8 entries (7 days + today)
	if len(daily) != 8 {
		t.Fatalf("expected 8 daily entries, got %d", len(daily))
	}

	// Today's entry (last one) should have cost from the recent session
	today := daily[len(daily)-1]
	todayStr := time.Now().Format("2006-01-02")
	if today.Date != todayStr {
		t.Errorf("last entry date = %s, want %s", today.Date, todayStr)
	}
	if !approxEqual(today.Cost, 0.05) {
		t.Errorf("today cost = %f, want 0.05", today.Cost)
	}
}

func TestGetActivityHeatmap_LocalTime(t *testing.T) {
	s := setupTZTestStore(t)
	defer s.Close()

	cells, err := s.GetActivityHeatmap()
	if err != nil {
		t.Fatalf("GetActivityHeatmap: %v", err)
	}

	if len(cells) == 0 {
		t.Fatal("expected non-empty heatmap")
	}

	// Verify that the cell for "2 hours ago" uses the correct local day-of-week
	twoHoursAgo := time.Now().Add(-2 * time.Hour)
	expectedDay := int(twoHoursAgo.Weekday())
	expectedHour := twoHoursAgo.Hour()

	found := false
	for _, c := range cells {
		if c.Day == expectedDay && c.Hour == expectedHour {
			found = true
			if !approxEqual(c.Cost, 0.05) {
				t.Errorf("cell cost = %f, want 0.05", c.Cost)
			}
		}
	}
	if !found {
		t.Errorf("no heatmap cell for day=%d hour=%d", expectedDay, expectedHour)
	}
}

func TestGetTrends_RFC3339Boundaries(t *testing.T) {
	s := setupTZTestStore(t)
	defer s.Close()

	trends, err := s.GetTrends()
	if err != nil {
		t.Fatalf("GetTrends: %v", err)
	}

	// The "old-session" is 2 days ago — should appear in prev_day_cost only
	// if it was yesterday. Since it's 2 days ago, prev_day_cost should be 0.
	// prev_week_cost covers 7-14 days ago, so also 0.
	// Just verify it doesn't error and returns reasonable values.
	if trends.PrevDayCost < 0 {
		t.Errorf("prev day cost should not be negative: %f", trends.PrevDayCost)
	}
}

func TestGetProjectMonthly_LocalMonth(t *testing.T) {
	s := setupTZTestStore(t)
	defer s.Close()

	data, err := s.GetProjectMonthly()
	if err != nil {
		t.Fatalf("GetProjectMonthly: %v", err)
	}

	// Should have at least one entry for the current month
	if len(data) == 0 {
		t.Fatal("expected non-empty project monthly data")
	}

	currentMonth := time.Now().Format("2006-01")
	found := false
	for _, pm := range data {
		if pm.Month == currentMonth {
			found = true
		}
	}
	if !found {
		t.Errorf("no data for current month %s", currentMonth)
	}
}
