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

const agentImage = "stok-agent:latest"

// DockerRuntime implements Runtime using the docker CLI.
type DockerRuntime struct{}

func (d *DockerRuntime) Name() string { return "docker" }

func (d *DockerRuntime) Available() bool {
	return commandExists("docker")
}

func (d *DockerRuntime) StartSession(ctx context.Context, id, projectPath, provider string) (Session, error) {
	worktreePath, gitRoot, err := CreateWorktree(projectPath, id)
	if err != nil {
		// Not a git repo or worktree failed — use a plain workspace dir.
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

	out, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return Session{}, fmt.Errorf("docker run: %w: %s", err, string(out))
	}
	return Session{ID: name, ProjectPath: worktreePath, Provider: provider, Status: "running", Runtime: "docker"}, nil
}

func (d *DockerRuntime) StopSession(ctx context.Context, id string) error {
	// Inspect the container to find the worktree path before removing it.
	inspectOut, _ := exec.CommandContext(ctx, "docker", "inspect",
		"--format", "{{range .Mounts}}{{.Source}}\n{{end}}", id).Output()
	out, err := exec.CommandContext(ctx, "docker", "rm", "-f", id).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker rm: %w: %s", err, string(out))
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

func (d *DockerRuntime) ListSessions(ctx context.Context) ([]Session, error) {
	out, err := exec.CommandContext(ctx, "docker", "ps", "-a",
		"--filter", "name=stok-session-",
		"--format", "{{.Names}}\t{{.Status}}",
	).Output()
	if err != nil {
		return nil, fmt.Errorf("docker ps: %w", err)
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
			Runtime: "docker",
		})
	}
	return sessions, nil
}

func (d *DockerRuntime) ExecAttach(ctx context.Context, id, provider, model string, cols, rows uint16) (*os.File, error) {
	providerArgs := providerArgs(provider, model)
	args := append([]string{
		"exec", "-it",
		"-e", "TERM=xterm-256color",
		"-e", "COLORTERM=truecolor",
		id,
	}, providerArgs...)
	cmd := exec.CommandContext(ctx, "docker", args...)
	f, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: cols, Rows: rows})
	if err != nil {
		return nil, fmt.Errorf("docker exec pty: %w", err)
	}
	return f, nil
}

// providerArgs returns the command slice for the given provider to pass to docker/container exec.
func providerArgs(provider, model string) []string {
	var args []string
	switch provider {
	case "claude":
		args = []string{"claude"}
		if model != "" {
			args = append(args, "--model", model)
		}
	case "gemini":
		args = []string{"gemini"}
		if model != "" {
			args = append(args, "--model", model)
		}
	case "copilot":
		args = []string{"gh", "copilot"}
	case "codex":
		args = []string{"codex"}
		if model != "" {
			args = append(args, "--model", model)
		}
	case "opencode":
		args = []string{"opencode"}
		if model != "" {
			args = append(args, "--model", model)
		}
	case "qwen":
		args = []string{"ollama", "run", "qwen3.5"}
	case "llama":
		args = []string{"ollama", "run", "llama3.2"}
	default:
		args = []string{"/bin/bash"}
	}
	return args
}
