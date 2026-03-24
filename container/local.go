package container

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
)

// LocalRuntime runs agent sessions as plain local processes (no container).
// Used as a fallback when Docker and Apple Container runtimes are unavailable.
type LocalRuntime struct {
	mu       sync.Mutex
	sessions map[string]Session
}

func NewLocalRuntime() *LocalRuntime {
	return &LocalRuntime{sessions: make(map[string]Session)}
}

func (l *LocalRuntime) Name() string { return "local" }

func (l *LocalRuntime) Available() bool { return true }

func (l *LocalRuntime) StartSession(_ context.Context, id, projectPath, provider string) (Session, error) {
	// Create a git worktree for this session so each AI agent works in isolation.
	worktreePath, _, err := CreateWorktree(projectPath, id)
	if err != nil {
		worktreePath = workspaceFallback(id)
	}
	copyDotEnv(projectPath, worktreePath)
	sess := Session{
		ID:          id,
		ProjectPath: worktreePath,
		Provider:    provider,
		Status:      "running",
		Runtime:     "local",
	}
	l.mu.Lock()
	l.sessions[id] = sess
	l.mu.Unlock()
	return sess, nil
}

func (l *LocalRuntime) StopSession(_ context.Context, id string) error {
	l.mu.Lock()
	sess, ok := l.sessions[id]
	delete(l.sessions, id)
	l.mu.Unlock()
	if ok {
		RemoveWorktree(sess.ProjectPath)
	}
	return nil
}

func (l *LocalRuntime) ListSessions(_ context.Context) ([]Session, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]Session, 0, len(l.sessions))
	for _, s := range l.sessions {
		out = append(out, s)
	}
	return out, nil
}

func (l *LocalRuntime) ExecAttach(_ context.Context, id, provider, model string, cols, rows uint16) (*os.File, error) {
	l.mu.Lock()
	sess, ok := l.sessions[id]
	l.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("session %s not found", id)
	}

	cmd := providerCommand(provider, model, sess.ProjectPath)
	f, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: cols, Rows: rows})
	if err != nil {
		return nil, fmt.Errorf("pty start: %w", err)
	}
	return f, nil
}

// providerCommand builds the exec.Cmd for the given provider.
func providerCommand(provider, model, projectPath string) *exec.Cmd {
	var cmd *exec.Cmd
	switch provider {
	case "claude":
		args := []string{}
		if model != "" {
			args = append(args, "--model", model)
		}
		cmd = exec.Command("claude", args...)
	case "gemini":
		args := []string{}
		if model != "" {
			args = append(args, "--model", model)
		}
		cmd = exec.Command("gemini", args...)
	case "codex":
		args := []string{}
		if model != "" {
			args = append(args, "--model", model)
		}
		cmd = exec.Command("codex", args...)
	case "copilot":
		cmd = exec.Command("gh", "copilot")
	case "opencode":
		args := []string{}
		if model != "" {
			args = append(args, "--model", model)
		}
		cmd = exec.Command("opencode", args...)
	default: // "shell" or anything else
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/zsh"
		}
		cmd = exec.Command(shell, "-l")
	}
	cmd.Dir = projectPath
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")
	return cmd
}
