package store

import (
	"testing"
)

// setupTestStore creates a store with a temp DB and one test session.
func setupTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	// Insert a test session so queries have data to iterate over
	err = s.UpsertSession(SessionDelta{
		ID:              "test-session-1",
		Project:         "test-project",
		Slug:            "test",
		Model:           "claude-sonnet-4-20250514",
		Timestamp:       "2026-03-19T10:00:00Z",
		DeltaInput:      100,
		DeltaOutput:     50,
		DeltaCacheRead:  10,
		DeltaCacheWrite: 5,
		DeltaCost:       0.01,
	})
	if err != nil {
		t.Fatalf("upsert session: %v", err)
	}

	return s
}

// These tests verify that rows.Err() checks don't break normal operation.
// The checks are defensive — they catch mid-iteration DB errors that are
// hard to trigger in tests. We verify the happy path still works.

func TestGetDailySummary_RowsErr(t *testing.T) {
	s := setupTestStore(t)
	defer s.Close()

	daily, err := s.GetDailySummary(30)
	if err != nil {
		t.Fatalf("GetDailySummary: %v", err)
	}
	if len(daily) == 0 {
		t.Error("expected non-empty daily summary")
	}
}

func TestTopSessions_RowsErr(t *testing.T) {
	s := setupTestStore(t)
	defer s.Close()

	sessions, err := s.TopSessions(10)
	if err != nil {
		t.Fatalf("TopSessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(sessions))
	}
}

func TestRecentSessions_RowsErr(t *testing.T) {
	s := setupTestStore(t)
	defer s.Close()

	sessions, err := s.RecentSessions(10)
	if err != nil {
		t.Fatalf("RecentSessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(sessions))
	}
}

func TestGetProjects_RowsErr(t *testing.T) {
	s := setupTestStore(t)
	defer s.Close()

	projects, err := s.GetProjects()
	if err != nil {
		t.Fatalf("GetProjects: %v", err)
	}
	if len(projects) != 1 {
		t.Errorf("expected 1 project, got %d", len(projects))
	}
}

func TestGetProjectMonthly_RowsErr(t *testing.T) {
	s := setupTestStore(t)
	defer s.Close()

	data, err := s.GetProjectMonthly()
	if err != nil {
		t.Fatalf("GetProjectMonthly: %v", err)
	}
	// May or may not have data depending on date functions, but should not error
	_ = data
}

func TestGetCostBreakdown_RowsErr(t *testing.T) {
	s := setupTestStore(t)
	defer s.Close()

	result, err := s.GetCostBreakdown()
	if err != nil {
		t.Fatalf("GetCostBreakdown: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestGetModelBreakdown_RowsErr(t *testing.T) {
	s := setupTestStore(t)
	defer s.Close()

	models, err := s.GetModelBreakdown()
	if err != nil {
		t.Fatalf("GetModelBreakdown: %v", err)
	}
	if len(models) != 1 {
		t.Errorf("expected 1 model, got %d", len(models))
	}
}

func TestGetActivityHeatmap_RowsErr(t *testing.T) {
	s := setupTestStore(t)
	defer s.Close()

	cells, err := s.GetActivityHeatmap()
	if err != nil {
		t.Fatalf("GetActivityHeatmap: %v", err)
	}
	// Should have at least one cell for the test session
	if len(cells) == 0 {
		t.Error("expected non-empty heatmap")
	}
}

func TestListSessions_RowsErr(t *testing.T) {
	s := setupTestStore(t)
	defer s.Close()

	sessions, total, err := s.ListSessions(10, 0, "cost", "desc")
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total=1, got %d", total)
	}
	if len(sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(sessions))
	}
}

func TestGetSessionRequests_RowsErr(t *testing.T) {
	s := setupTestStore(t)
	defer s.Close()

	// Insert a request record
	err := s.UpsertRequest(RequestRecord{
		RequestID:    "req-1",
		SessionID:    "test-session-1",
		Timestamp:    "2026-03-19T10:00:00Z",
		Model:        "claude-sonnet-4-20250514",
		InputTokens:  100,
		OutputTokens: 50,
		Cost:         0.01,
	})
	if err != nil {
		t.Fatalf("upsert request: %v", err)
	}

	recs, err := s.GetSessionRequests("test-session-1")
	if err != nil {
		t.Fatalf("GetSessionRequests: %v", err)
	}
	if len(recs) != 1 {
		t.Errorf("expected 1 request, got %d", len(recs))
	}
}
