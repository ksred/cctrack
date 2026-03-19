package parser

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// DiscoverFiles walks the log directory and returns all .jsonl files.
// Symlinks are skipped to prevent reading unintended files.
func DiscoverFiles(logDir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(logDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible dirs
		}
		// Skip symlinks to prevent reading unintended files
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(path, ".jsonl") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// ExtractSessionInfo derives session ID and project name from a JSONL file path.
// Paths look like:
//
//	~/.claude/projects/-home-user-Github-project/SESSION_UUID.jsonl
//	~/.claude/projects/-home-user-Github-project/SESSION_UUID/subagents/agent-XXXX.jsonl
func ExtractSessionInfo(path string) SessionInfo {
	info := SessionInfo{}

	dir := filepath.Dir(path)
	base := filepath.Base(path)

	// Check if this is a subagent file
	if strings.Contains(path, "/subagents/") {
		info.IsSubagent = true
		// Walk up to find the session directory
		// .../SESSION_UUID/subagents/agent-xxx.jsonl
		parts := strings.Split(path, "/subagents/")
		if len(parts) > 0 {
			sessionDir := parts[0]
			info.SessionID = filepath.Base(sessionDir)
			info.Project = extractProject(filepath.Dir(sessionDir))
		}
		return info
	}

	// Regular session file: .../PROJECT_DIR/SESSION_UUID.jsonl
	info.SessionID = strings.TrimSuffix(base, ".jsonl")
	info.Project = extractProject(dir)
	return info
}

func extractProject(dir string) string {
	base := filepath.Base(dir)
	// Project dirs look like: -home-user-Github-my-project
	// We want to reconstruct the actual directory path and return
	// everything after common parent dirs (home, username, Github, etc.)
	if strings.HasPrefix(base, "-") {
		// Convert dash-separated encoding back to path segments
		segments := strings.Split(base[1:], "-")

		// Find the last "anchor" directory (Github, Projects, repos, src, etc.)
		// and return everything after it as the project name
		anchors := map[string]bool{
			"Github": true, "github": true,
			"Projects": true, "projects": true,
			"repos": true, "Repos": true,
			"src": true, "code": true, "Code": true,
			"workspace": true, "Workspace": true,
			"worktrees": true,
		}

		for i, seg := range segments {
			if anchors[seg] && i+1 < len(segments) {
				// Join everything after the anchor with hyphens
				return strings.Join(segments[i+1:], "-")
			}
		}

		// No anchor found — skip home/username and return the rest
		if len(segments) > 2 {
			return strings.Join(segments[2:], "-")
		}
	}
	return base
}
