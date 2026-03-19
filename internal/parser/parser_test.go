package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ksred/cctrack/internal/store"
)

func TestParseFile_SkipsNegativeTokens(t *testing.T) {
	s, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer s.Close()

	p := New(s)

	// Create a JSONL file with one valid event and one with negative tokens
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "-home-user-Github-test")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	jsonl := `{"type":"assistant","sessionId":"sess-valid","timestamp":"2026-03-19T10:00:00Z","message":{"model":"claude-sonnet-4-20250514","usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}
{"type":"assistant","sessionId":"sess-negative","timestamp":"2026-03-19T10:01:00Z","message":{"model":"claude-sonnet-4-20250514","usage":{"input_tokens":-100,"output_tokens":50,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`

	testFile := filepath.Join(projectDir, "test-session.jsonl")
	if err := os.WriteFile(testFile, []byte(jsonl), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	sessions, err := p.ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	// Only the valid session should be affected
	if len(sessions) != 1 {
		t.Errorf("expected 1 affected session, got %d", len(sessions))
	}

	// The negative-token session should not exist in the store
	_, err = s.GetSession("sess-negative")
	if err == nil {
		t.Error("sess-negative should not exist in store (negative tokens should be skipped)")
	}

	// The valid session should exist
	sess, err := s.GetSession("sess-valid")
	if err != nil {
		t.Fatalf("sess-valid should exist: %v", err)
	}
	if sess.TotalInput != 100 {
		t.Errorf("expected input=100, got %d", sess.TotalInput)
	}
}
