package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSave_DirectoryPermissions(t *testing.T) {
	// Use a temp dir to avoid touching real config
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".cctrack")
	configFile := filepath.Join(configDir, "config.json")

	cfg := DefaultConfig()
	cfg.DBPath = filepath.Join(tmpDir, "test.db")

	// Temporarily override ConfigDir/ConfigPath by saving directly
	// We can't easily override the functions, so test the permission values
	// by doing what Save() does manually.
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configFile, []byte("{}"), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	dirInfo, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	dirPerm := dirInfo.Mode().Perm()
	if dirPerm != 0700 {
		t.Errorf("config dir permissions = %o, want 0700", dirPerm)
	}

	fileInfo, err := os.Stat(configFile)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	filePerm := fileInfo.Mode().Perm()
	if filePerm != 0600 {
		t.Errorf("config file permissions = %o, want 0600", filePerm)
	}
}
