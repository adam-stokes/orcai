package chatui

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// geminiRecord is one entry in ~/.stok/sessions/{encoded-cwd}.json.
type geminiRecord struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Prompt    string    `json:"prompt"` // first 60 chars of the initial prompt
}

// stokSessionsDir returns ~/.stok/sessions/
func stokSessionsDir(homeDir string) string {
	return filepath.Join(homeDir, ".stok", "sessions")
}

// geminiRegistryPath returns the path to the Gemini session registry for the given cwd.
func geminiRegistryPath(cwd, homeDir string) string {
	encoded := strings.ReplaceAll(cwd, "/", "-")
	return filepath.Join(stokSessionsDir(homeDir), encoded+".json")
}

// loadGeminiRegistry reads the registry file; returns nil slice if missing.
func loadGeminiRegistry(cwd, homeDir string) []geminiRecord {
	data, err := os.ReadFile(geminiRegistryPath(cwd, homeDir))
	if err != nil {
		return nil
	}
	var records []geminiRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil
	}
	return records
}

// saveGeminiSession appends or updates a Gemini session in the registry.
// prompt should be the first 60 chars of the user's initial prompt.
func saveGeminiSession(cwd, homeDir, sessionID, prompt string) {
	if sessionID == "" {
		return
	}
	records := loadGeminiRegistry(cwd, homeDir)
	now := time.Now().UTC()

	// Update existing record if session ID already present.
	for i, r := range records {
		if r.ID == sessionID {
			records[i].UpdatedAt = now
			writeGeminiRegistry(cwd, homeDir, records)
			return
		}
	}

	// Append new record.
	p := prompt
	if len(p) > 60 {
		p = p[:60]
	}
	records = append(records, geminiRecord{
		ID:        sessionID,
		CreatedAt: now,
		UpdatedAt: now,
		Prompt:    p,
	})
	writeGeminiRegistry(cwd, homeDir, records)
}

func writeGeminiRegistry(cwd, homeDir string, records []geminiRecord) {
	dir := stokSessionsDir(homeDir)
	_ = os.MkdirAll(dir, 0755)
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(geminiRegistryPath(cwd, homeDir), data, 0644)
}

// loadClaudeConvHistory reads the .jsonl session file for the given session ID
// and extracts user/assistant turns to display when resuming a session.
func loadClaudeConvHistory(cwd, homeDir, sessionID string) []message {
	encoded := strings.ReplaceAll(cwd, "/", "-")
	dir := filepath.Join(homeDir, ".claude", "projects", encoded)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	// Find the .jsonl file that contains the session ID.
	var filePath string
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		fPath := filepath.Join(dir, e.Name())
		f, err := os.Open(fPath)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 512*1024), 512*1024)
		found := false
		for scanner.Scan() {
			var obj struct {
				SessionID string `json:"sessionId"`
			}
			if json.Unmarshal(scanner.Bytes(), &obj) == nil && obj.SessionID == sessionID {
				found = true
				break
			}
		}
		_ = f.Close()
		if found {
			filePath = fPath
			break
		}
	}
	if filePath == "" {
		return nil
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer f.Close()

	var messages []message
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		var obj struct {
			Message struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal(scanner.Bytes(), &obj) != nil {
			continue
		}
		msgRole := obj.Message.Role
		if msgRole == "" {
			continue
		}

		// Content may be a plain string or an array of content blocks.
		var content string
		if json.Unmarshal(obj.Message.Content, &content) != nil {
			var blocks []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}
			if json.Unmarshal(obj.Message.Content, &blocks) == nil {
				var sb strings.Builder
				for _, b := range blocks {
					if b.Type == "text" {
						sb.WriteString(b.Text)
					}
				}
				content = sb.String()
			}
		}
		if content == "" {
			continue
		}

		switch msgRole {
		case "user":
			messages = append(messages, message{role: roleUser, content: content})
		case "assistant":
			messages = append(messages, message{role: roleAssistant, content: content})
		}
	}
	return messages
}

// loadGeminiConvHistory reads the stored conversation history for a Gemini session.
func loadGeminiConvHistory(cwd, homeDir, sessionID string) []message {
	encoded := strings.ReplaceAll(cwd, "/", "-")
	path := filepath.Join(homeDir, ".stok", "sessions", "history", encoded, sessionID+".json")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var convMsgs []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	if json.Unmarshal(data, &convMsgs) != nil {
		return nil
	}
	messages := make([]message, 0, len(convMsgs))
	for _, m := range convMsgs {
		switch m.Role {
		case "user":
			messages = append(messages, message{role: roleUser, content: m.Content})
		case "assistant":
			messages = append(messages, message{role: roleAssistant, content: m.Content})
		}
	}
	return messages
}

// loadCopilotSessions scans ~/.copilot/session-state/ and returns sessions
// whose cwd matches the given working directory.
func loadCopilotSessions(cwd, homeDir string) []sessionEntry {
	stateDir := filepath.Join(homeDir, ".copilot", "session-state")
	entries, err := os.ReadDir(stateDir)
	if err != nil {
		return nil
	}

	var sessions []sessionEntry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		wsPath := filepath.Join(stateDir, e.Name(), "workspace.yaml")
		data, err := os.ReadFile(wsPath)
		if err != nil {
			continue
		}
		id, wsCwd, updatedAtStr := parseCopilotWorkspace(data)
		if id == "" || wsCwd != cwd {
			continue
		}
		updatedAt, _ := time.Parse(time.RFC3339Nano, updatedAtStr)
		displayName := id
		if len(displayName) > 8 {
			displayName = displayName[:8]
		}
		sessions = append(sessions, sessionEntry{
			id:        id,
			name:      displayName,
			provider:  "copilot",
			updatedAt: updatedAt,
		})
	}
	return sessions
}

// parseCopilotWorkspace extracts id, cwd, and updated_at from workspace.yaml bytes.
func parseCopilotWorkspace(data []byte) (id, cwd, updatedAt string) {
	for _, line := range strings.Split(string(data), "\n") {
		k, v, ok := strings.Cut(line, ": ")
		if !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "id":
			id = strings.TrimSpace(v)
		case "cwd":
			cwd = strings.TrimSpace(v)
		case "updated_at":
			updatedAt = strings.TrimSpace(v)
		}
	}
	return
}
