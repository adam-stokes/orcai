package switchboard

import (
	"fmt"
	"os/exec"

	"github.com/adam-stokes/orcai/internal/themes"
)

// applyTmuxTheme pushes theme colors to the running tmux session via set-option.
// Called after the user selects a new theme so the status bar updates immediately
// without needing a session restart.
func applyTmuxTheme(b *themes.Bundle) {
	if b == nil {
		return
	}
	accent := b.Palette.Accent
	bg := b.Palette.BG
	dim := b.Palette.Dim
	border := b.Palette.Border

	// Fall back to Nord defaults if palette fields are empty.
	if accent == "" {
		accent = "#88c0d0"
	}
	if bg == "" {
		bg = "#2e3440"
	}
	if dim == "" {
		dim = "#4c566a"
	}
	if border == "" {
		border = "#3b4252"
	}

	opts := [][]string{
		{"set-option", "-g", "status-style", fmt.Sprintf("fg=%s,bg=%s", accent, bg)},
		{"set-option", "-g", "status-left", fmt.Sprintf("#[fg=%s,bold] ORCAI #[default]", accent)},
		{"set-option", "-g", "status-right-length", "200"},
		{"set-option", "-g", "status-right", fmt.Sprintf("#[fg=%s] ^spc h help  ^spc t switchboard  ^spc m themes  ^spc j jump  ^spc c win  ^spc d detach  ^spc r reload  ^spc q quit  %%H:%%M ", dim)},
		{"set-option", "-g", "pane-border-style", fmt.Sprintf("fg=%s", border)},
		{"set-option", "-g", "pane-active-border-style", fmt.Sprintf("fg=%s", accent)},
	}

	for _, args := range opts {
		_ = exec.Command("tmux", args...).Run()
	}
}
