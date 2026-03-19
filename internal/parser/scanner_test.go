package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverFiles_SkipsSymlinks(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a real .jsonl file
	realFile := filepath.Join(tmpDir, "real.jsonl")
	if err := os.WriteFile(realFile, []byte(`{}`), 0644); err != nil {
		t.Fatalf("write real file: %v", err)
	}

	// Create a symlinked .jsonl file pointing outside
	symFile := filepath.Join(tmpDir, "symlink.jsonl")
	os.Symlink("/etc/passwd", symFile)

	files, err := DiscoverFiles(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverFiles: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("expected 1 file (real only), got %d: %v", len(files), files)
	}
	if len(files) == 1 && files[0] != realFile {
		t.Errorf("expected %s, got %s", realFile, files[0])
	}
}

func TestDiscoverFiles_SkipsSymlinkedDirs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a directory with a .jsonl file outside the tree
	outsideDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(outsideDir, "outside.jsonl"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Symlink that directory into our scan root
	symDir := filepath.Join(tmpDir, "linked-dir")
	os.Symlink(outsideDir, symDir)

	// Create a real file in the root
	realFile := filepath.Join(tmpDir, "real.jsonl")
	if err := os.WriteFile(realFile, []byte(`{}`), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	files, err := DiscoverFiles(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverFiles: %v", err)
	}

	// Should only find the real file, not the one inside the symlinked dir
	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d: %v", len(files), files)
	}
}

func TestDiscoverFiles_NormalDirs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a subdirectory with a .jsonl file (no symlinks)
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "nested.jsonl"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "root.jsonl"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	files, err := DiscoverFiles(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverFiles: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d: %v", len(files), files)
	}
}
