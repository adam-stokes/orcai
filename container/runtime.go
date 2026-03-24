package container

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Session represents a running container workspace for an agent session.
type Session struct {
	ID          string `json:"id"`
	ProjectPath string `json:"projectPath"`
	Provider    string `json:"provider"` // "claude" | "gemini" | "copilot" | "codex" | "opencode" | "qwen" | "llama" | "shell"
	Model       string `json:"model"`    // optional model override (e.g. "claude-opus-4-6", "gemini-2.5-pro")
	Status      string `json:"status"`   // "running" | "stopped"
	Runtime     string `json:"runtime"`  // "docker" | "apple" | "local"
}

// Runtime abstracts container lifecycle operations.
// Implementations: DockerRuntime, AppleRuntime.
type Runtime interface {
	Name() string
	Available() bool
	StartSession(ctx context.Context, id, projectPath, provider string) (Session, error)
	StopSession(ctx context.Context, id string) error
	ListSessions(ctx context.Context) ([]Session, error)
	// ExecAttach opens an interactive PTY to the container's shell.
	// Returns the PTY master file; caller is responsible for closing it.
	ExecAttach(ctx context.Context, id, provider, model string, cols, rows uint16) (*os.File, error)
}

// ContainerName returns the container name for a session ID.
func ContainerName(sessionID string) string {
	return "stok-session-" + sessionID
}

// DetectAvailableProviders returns the provider IDs whose CLIs are installed.
// "shell" is always included.
func DetectAvailableProviders() []string {
	available := []string{}
	if commandExists("claude") {
		available = append(available, "claude")
	}
	if commandExists("gemini") {
		available = append(available, "gemini")
	}
	if commandExists("gh") {
		if exec.Command("gh", "copilot", "--version").Run() == nil {
			available = append(available, "copilot")
		}
	}
	if commandExists("codex") {
		available = append(available, "codex")
	}
	if commandExists("opencode") {
		available = append(available, "opencode")
	}
	available = append(available, "shell")
	return available
}

// Detect returns the best available container runtime.
// Prefers Apple Containers on darwin when the system is running,
// then Docker, then falls back to a local shell runtime.
func Detect() Runtime {
	if isAppleRunning() {
		return &AppleRuntime{}
	}
	if isDockerRunning() {
		return &DockerRuntime{}
	}
	return NewLocalRuntime()
}

func isAppleRunning() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	if _, err := exec.LookPath("container"); err != nil {
		return false
	}
	// Probe: list images — fails if the container system services aren't started.
	err := exec.Command("container", "images", "--format", "{{.ID}}").Run()
	return err == nil
}

func isDockerRunning() bool {
	if _, err := exec.LookPath("docker"); err != nil {
		return false
	}
	if err := exec.Command("docker", "info").Run(); err != nil {
		return false
	}
	// Only use Docker if the stok-agent image is available.
	out, err := exec.Command("docker", "images", "-q", agentImage).Output()
	return err == nil && len(strings.TrimSpace(string(out))) > 0
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// CreateWorktree creates a git worktree at ~/.stok/worktrees/{sessionID}.
// Returns (worktreePath, gitRoot, nil) on success.
func CreateWorktree(repoRoot, sessionID string) (worktreePath, gitRoot string, err error) {
	out, err := exec.Command("git", "-C", repoRoot, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", "", fmt.Errorf("not a git repo: %w", err)
	}
	gitRoot = strings.TrimSpace(string(out))

	homeDir, _ := os.UserHomeDir()
	worktreePath = filepath.Join(homeDir, ".stok", "worktrees", sessionID)
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0o755); err != nil {
		return "", "", err
	}

	branch := "stok-session-" + sessionID
	cmd := exec.Command("git", "-C", gitRoot, "worktree", "add", "-b", branch, worktreePath)
	if outBytes, err := cmd.CombinedOutput(); err != nil {
		return "", "", fmt.Errorf("git worktree add: %w: %s", err, string(outBytes))
	}

	trustPath(worktreePath)
	return worktreePath, gitRoot, nil
}

// workspaceFallback returns a plain (non-git) workspace directory for a session.
// Used when the project path is not a git repo or worktree creation fails.
func workspaceFallback(sessionID string) string {
	homeDir, _ := os.UserHomeDir()
	path := filepath.Join(homeDir, ".stok", "workspaces", sessionID)
	_ = os.MkdirAll(path, 0o755)
	trustPath(path)
	return path
}

// DetectOllamaModels returns the names of locally installed Ollama models.
// Returns nil if Ollama is not running or returns an error.
func DetectOllamaModels() []string {
	resp, err := http.Get("http://localhost:11434/api/tags")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}
	names := make([]string, 0, len(result.Models))
	for _, m := range result.Models {
		names = append(names, m.Name)
	}
	return names
}

// RemoveWorktree removes the git worktree at worktreePath.
func RemoveWorktree(worktreePath string) {
	exec.Command("git", "worktree", "remove", "--force", worktreePath).Run()
	os.RemoveAll(worktreePath)
}

// copyDotEnv copies .env from src directory to dst if it exists.
// Gitignored files like .env are not included in worktrees, so we copy them manually.
func copyDotEnv(src, dst string) {
	data, err := os.ReadFile(filepath.Join(src, ".env"))
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(dst, ".env"), data, 0o600)
}

// trustPath marks path as trusted for all AI CLIs that support folder trust.
// Currently handles Gemini (~/.gemini/trustedFolders.json).
func trustPath(path string) {
	trustGemini(path)
	trustCopilot(path)
}

// trustCopilot adds path to ~/.copilot/config.json trusted_folders.
func trustCopilot(path string) {
	homeDir, _ := os.UserHomeDir()
	jsonPath := filepath.Join(homeDir, ".copilot", "config.json")

	var cfg map[string]interface{}
	if data, err := os.ReadFile(jsonPath); err == nil {
		_ = json.Unmarshal(data, &cfg)
	}
	if cfg == nil {
		cfg = make(map[string]interface{})
	}

	var folders []string
	if raw, ok := cfg["trusted_folders"]; ok {
		if arr, ok := raw.([]interface{}); ok {
			for _, v := range arr {
				if s, ok := v.(string); ok {
					if s == path {
						return // already trusted
					}
					folders = append(folders, s)
				}
			}
		}
	}
	folders = append(folders, path)
	cfg["trusted_folders"] = folders

	data, _ := json.MarshalIndent(cfg, "", "  ")
	_ = os.WriteFile(jsonPath, data, 0o644)
}

// trustGemini adds path to ~/.gemini/trustedFolders.json with TRUST_FOLDER.
// Gemini's format: {"<path>": "TRUST_FOLDER"|"TRUST_PARENT", ...}
func trustGemini(path string) {
	homeDir, _ := os.UserHomeDir()
	jsonPath := filepath.Join(homeDir, ".gemini", "trustedFolders.json")

	trusted := make(map[string]string)
	if data, err := os.ReadFile(jsonPath); err == nil {
		_ = json.Unmarshal(data, &trusted)
	}
	if _, ok := trusted[path]; ok {
		return // already trusted
	}
	trusted[path] = "TRUST_FOLDER"
	data, _ := json.MarshalIndent(trusted, "", "  ")
	_ = os.WriteFile(jsonPath, data, 0o644)
}
