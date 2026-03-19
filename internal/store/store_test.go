package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpen_DirectoryPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	dbDir := filepath.Join(tmpDir, "newdir")
	dbPath := filepath.Join(dbDir, "test.db")

	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	s.Close()

	info, err := os.Stat(dbDir)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0700 {
		t.Errorf("db dir permissions = %o, want 0700", perm)
	}
}
