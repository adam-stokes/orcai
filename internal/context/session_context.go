package context

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	maxSessions = 3
	maxMessages = 20
)

type sessionMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// GetRecentContext returns up to maxMessages recent messages from the
// maxSessions most recent stok session files for the given cwd.
// Returns an empty (non-nil) slice if no sessions exist.
// Messages are formatted as "role: content".
func GetRecentContext(homeDir, cwd string) []string {
	encoded := strings.ReplaceAll(cwd, "/", "-")
	dir := filepath.Join(homeDir, ".stok", "sessions", "history", encoded)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return []string{}
	}

	type fileEntry struct {
		path    string
		modTime int64
	}
	var files []fileEntry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, fileEntry{
			path:    filepath.Join(dir, e.Name()),
			modTime: info.ModTime().Unix(),
		})
	}

	// Sort newest first.
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime > files[j].modTime
	})

	var out []string
	for i, f := range files {
		if i >= maxSessions {
			break
		}
		msgs := readSessionFile(f.path)
		// Take only the last maxMessages messages from this session.
		if len(msgs) > maxMessages {
			msgs = msgs[len(msgs)-maxMessages:]
		}
		for _, m := range msgs {
			if m.Role != "" && m.Content != "" {
				out = append(out, m.Role+": "+m.Content)
			}
		}
	}
	return out
}

func readSessionFile(path string) []sessionMsg {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var msgs []sessionMsg
	if err := json.Unmarshal(data, &msgs); err != nil {
		return nil
	}
	return msgs
}
