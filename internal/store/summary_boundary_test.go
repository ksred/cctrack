package store_test

import (
	"testing"
	"time"
)

// These tests pin the LOCAL-day boundary semantics of GetSummary (today /
// month buckets) and GetTrends (prev-day / prev-month) so the aggregation
// queries can be rewritten for index range scans without silently shifting
// a request across a midnight or month boundary. A request at 23:30 local
// belongs to that local day even when its UTC timestamp falls on the next
// calendar date — the recurring bug class this repo's F1 tests guard in
// GetDailySummary, extended here to the summary/trends surfaces.

func localMidnight(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

func TestGetSummary_TodayBucket_LocalMidnightBoundary(t *testing.T) {
	s := newTestStore(t)
	f := newFixtureBuilder(t, s)

	todayMid := localMidnight(time.Now())

	// 23:30 local yesterday — must NOT count toward today.
	f.ingest("sess-yesterday", "p", todayMid.Add(-30*time.Minute), 10.00, 1000, 500)
	// 00:30 local today — must count toward today.
	f.ingest("sess-today", "p", todayMid.Add(30*time.Minute), 3.00, 300, 150)

	sum, err := s.GetSummary()
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	if !floatEq(sum.Today.Cost, 3.00) {
		t.Errorf("Today.Cost = %v, want 3.00 (only the post-local-midnight request)", sum.Today.Cost)
	}
	if sum.Today.Tokens != 450 {
		t.Errorf("Today.Tokens = %v, want 450", sum.Today.Tokens)
	}
}

func TestGetSummary_MonthBucket_LocalMonthBoundary(t *testing.T) {
	s := newTestStore(t)
	f := newFixtureBuilder(t, s)

	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)

	// 23:30 local on the last day of the previous month — excluded.
	f.ingest("sess-prev-month", "p", monthStart.Add(-30*time.Minute), 20.00, 2000, 1000)
	// 00:30 local on the 1st — included.
	f.ingest("sess-this-month", "p", monthStart.Add(30*time.Minute), 5.00, 500, 250)

	sum, err := s.GetSummary()
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	if !floatEq(sum.Month.Cost, 5.00) {
		t.Errorf("Month.Cost = %v, want 5.00 (only the post-month-start request)", sum.Month.Cost)
	}
}

func TestGetTrends_PrevDay_LocalDayBoundaries(t *testing.T) {
	s := newTestStore(t)
	f := newFixtureBuilder(t, s)

	todayMid := localMidnight(time.Now())
	yesterdayMid := todayMid.AddDate(0, 0, -1)

	// 23:30 local two days ago — before yesterday, excluded.
	f.ingest("sess-d2", "p", yesterdayMid.Add(-30*time.Minute), 100.00, 1000, 500)
	// 00:30 and 23:30 local yesterday — both included.
	f.ingest("sess-d1a", "p", yesterdayMid.Add(30*time.Minute), 7.00, 700, 350)
	f.ingest("sess-d1b", "p", todayMid.Add(-30*time.Minute), 2.00, 200, 100)
	// 00:30 local today — after yesterday, excluded.
	f.ingest("sess-d0", "p", todayMid.Add(30*time.Minute), 50.00, 500, 250)

	trends, err := s.GetTrends()
	if err != nil {
		t.Fatalf("GetTrends: %v", err)
	}
	if !floatEq(trends.PrevDayCost, 9.00) {
		t.Errorf("PrevDayCost = %v, want 9.00 (both local-yesterday requests, nothing else)", trends.PrevDayCost)
	}
}

func TestGetTrends_PrevMonth_LocalMonthBoundaries(t *testing.T) {
	s := newTestStore(t)
	f := newFixtureBuilder(t, s)

	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	prevMonthStart := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, time.Local)

	// 23:30 local before the previous month began — excluded.
	f.ingest("sess-m2", "p", prevMonthStart.Add(-30*time.Minute), 100.00, 1000, 500)
	// Mid previous month and 23:30 local on its last day — both included.
	f.ingest("sess-m1a", "p", prevMonthStart.AddDate(0, 0, 14), 11.00, 1100, 550)
	f.ingest("sess-m1b", "p", monthStart.Add(-30*time.Minute), 4.00, 400, 200)
	// 00:30 local on the 1st of this month — excluded.
	f.ingest("sess-m0", "p", monthStart.Add(30*time.Minute), 50.00, 500, 250)

	trends, err := s.GetTrends()
	if err != nil {
		t.Fatalf("GetTrends: %v", err)
	}
	if !floatEq(trends.PrevMonthCost, 15.00) {
		t.Errorf("PrevMonthCost = %v, want 15.00 (both prev-month requests, nothing else)", trends.PrevMonthCost)
	}
}
