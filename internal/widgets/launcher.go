package widgets

import (
	"fmt"
	"os/exec"
)

// Launch opens a new tmux window named after the widget in the given tmux
// session and runs the widget binary inside it. The widget binary is
// responsible for connecting to the busd socket and registering itself.
//
// Returns an error if the tmux command itself fails to execute. Note that tmux
// may return success even if the widget binary is not found — the shell inside
// the tmux window will handle that error.
func Launch(m Manifest, tmuxSession string) error {
	cmd := exec.Command("tmux", "new-window",
		"-t", tmuxSession,
		"-n", m.Name,
		m.Binary,
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("widgets: launch %s in tmux session %s: %w", m.Name, tmuxSession, err)
	}
	return nil
}

// ResolveOverride returns the binary to use for the given widget manifest.
// If an orcai-<name> override binary is found in PATH, it is returned.
// Otherwise, the manifest's own Binary field is returned.
func ResolveOverride(m Manifest) string {
	if override, err := exec.LookPath("orcai-" + m.Name); err == nil {
		return override
	}
	return m.Binary
}

// LaunchWithOverride is like Launch but checks for an orcai-<name> PATH
// override before using the binary declared in the manifest.
func LaunchWithOverride(m Manifest, tmuxSession string) error {
	binary := ResolveOverride(m)
	cmd := exec.Command("tmux", "new-window",
		"-t", tmuxSession,
		"-n", m.Name,
		binary,
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("widgets: launch %s in tmux session %s: %w", m.Name, tmuxSession, err)
	}
	return nil
}
