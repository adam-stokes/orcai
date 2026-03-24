package container

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/creack/pty"
)

// AppleRuntime implements Runtime using Apple Containers (macOS 26+).
// Uses the 'container' CLI which mirrors the Docker CLI interface.
type AppleRuntime struct{}

func (a *AppleRuntime) Name() string { return "apple" }

func (a *AppleRuntime) Available() bool {
	return commandExists("container")
}

func (a *AppleRuntime) StartSession(ctx context.Context, id, projectPath, provider string) (Session, error) {
	worktreePath, gitRoot, err := CreateWorktree(projectPath, id)
	if err != nil {
		worktreePath = workspaceFallback(id)
		gitRoot = ""
	}

	homeDir, _ := os.UserHomeDir()
	name := ContainerName(id)

	args := []string{
		"run", "-d",
		"--name", name,
		"-v", worktreePath + ":" + worktreePath,
	}
	if gitRoot != "" {
		args = append(args, "-v", gitRoot+"/.git:"+gitRoot+"/.git")
	}
	args = append(args,
		"-v", homeDir+"/.claude:/root/.claude",
		"-v", homeDir+"/.claude.json:/root/.claude.json",
		"-v", homeDir+"/.gemini:/root/.gemini",
		"-v", homeDir+"/.config/gh:/root/.config/gh",
		"-v", homeDir+"/.ollama:/root/.ollama",
		"-w", worktreePath,
		agentImage,
	)

	out, err := exec.CommandContext(ctx, "container", args...).CombinedOutput()
	if err != nil {
		return Session{}, fmt.Errorf("container run: %w: %s", err, string(out))
	}
	return Session{ID: name, ProjectPath: worktreePath, Provider: provider, Status: "running", Runtime: "apple"}, nil
}

func (a *AppleRuntime) StopSession(ctx context.Context, id string) error {
	// Inspect the container to find the worktree path before removing it.
	inspectOut, _ := exec.CommandContext(ctx, "container", "inspect",
		"--format", "{{range .Mounts}}{{.Source}}\n{{end}}", id).Output()
	out, err := exec.CommandContext(ctx, "container", "rm", "-f", id).CombinedOutput()
	if err != nil {
		return fmt.Errorf("container rm: %w: %s", err, string(out))
	}
	// Clean up any managed worktrees that were mounted into this container.
	homeDir, _ := os.UserHomeDir()
	worktreeBase := filepath.Join(homeDir, ".stok", "worktrees")
	for _, line := range strings.Split(strings.TrimSpace(string(inspectOut)), "\n") {
		src := strings.TrimSpace(line)
		if strings.HasPrefix(src, worktreeBase) {
			RemoveWorktree(src)
		}
	}
	return nil
}

func (a *AppleRuntime) ListSessions(ctx context.Context) ([]Session, error) {
	out, err := exec.CommandContext(ctx, "container", "ps", "-a",
		"--filter", "name=stok-session-",
		"--format", "{{.Names}}\t{{.Status}}",
	).Output()
	if err != nil {
		return nil, fmt.Errorf("container ps: %w", err)
	}
	var sessions []Session
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		status := "stopped"
		if len(parts) > 1 && strings.HasPrefix(parts[1], "Up") {
			status = "running"
		}
		sessions = append(sessions, Session{
			ID:      parts[0],
			Status:  status,
			Runtime: "apple",
		})
	}
	return sessions, nil
}

func (a *AppleRuntime) ExecAttach(ctx context.Context, id, provider, model string, cols, rows uint16) (*os.File, error) {
	pArgs := providerArgs(provider, model)
	args := append([]string{
		"exec", "-it",
		"-e", "TERM=xterm-256color",
		"-e", "COLORTERM=truecolor",
		id,
	}, pArgs...)
	cmd := exec.CommandContext(ctx, "container", args...)
	f, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: cols, Rows: rows})
	if err != nil {
		return nil, fmt.Errorf("container exec pty: %w", err)
	}
	return f, nil
}
