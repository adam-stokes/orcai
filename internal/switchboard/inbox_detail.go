package switchboard

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/adam-stokes/orcai/internal/store"
)

type runMeta struct {
	PipelineFile string `json:"pipeline_file"`
	CWD          string `json:"cwd"`
}

func parseRunMetadata(raw string) runMeta {
	if raw == "" {
		return runMeta{}
	}
	var m runMeta
	_ = json.Unmarshal([]byte(raw), &m)
	return m
}

func collapseTilde(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

// buildRunContent formats the full detail text for a run, mirroring the
// content builder that was previously in internal/inbox/modal.go.
func buildRunContent(run store.Run, mc modalColors, markdownMode bool) string {
	dim := lipgloss.NewStyle().Foreground(mc.dim)
	fg := lipgloss.NewStyle().Foreground(mc.fg)
	success := lipgloss.NewStyle().Foreground(lipgloss.Color("#50fa7b"))
	errStyle := lipgloss.NewStyle().Foreground(mc.error)

	var sb strings.Builder

	// Started / finished / duration / exit status
	startedStr := time.UnixMilli(run.StartedAt).Format("2006-01-02 3:04:05 PM")
	sb.WriteString(dim.Render("started:  ") + fg.Render(startedStr) + "\n")

	if run.FinishedAt != nil {
		finishedStr := time.UnixMilli(*run.FinishedAt).Format("2006-01-02 3:04:05 PM")
		sb.WriteString(dim.Render("finished: ") + fg.Render(finishedStr) + "\n")

		dur := time.Duration((*run.FinishedAt - run.StartedAt) * int64(time.Millisecond))
		durationStr := dur.Round(time.Second).String()
		sb.WriteString(dim.Render("duration: ") + fg.Render(durationStr) + "  ")
	} else {
		sb.WriteString(dim.Render("finished: ") + fg.Render("(in progress)") + "\n")
		dur := time.Since(time.UnixMilli(run.StartedAt))
		sb.WriteString(dim.Render("duration: ") + fg.Render(dur.Round(time.Second).String()) + "  ")
	}

	if run.ExitStatus != nil {
		if *run.ExitStatus == 0 {
			sb.WriteString(dim.Render("exit: ") + success.Render("OK"))
		} else {
			sb.WriteString(dim.Render("exit: ") + errStyle.Render(fmt.Sprintf("ERROR (%d)", *run.ExitStatus)))
		}
	} else {
		sb.WriteString(dim.Render("exit: ") + fg.Render("(running)"))
	}
	sb.WriteString("\n")

	// Metadata: pipeline file and cwd
	meta := parseRunMetadata(run.Metadata)
	if meta.PipelineFile != "" {
		sb.WriteString(dim.Render("pipeline: ") + fg.Render(collapseTilde(meta.PipelineFile)) + "\n")
	}
	if meta.CWD != "" {
		sb.WriteString(dim.Render("cwd:      ") + fg.Render(collapseTilde(meta.CWD)) + "\n")
	}

	// Separator
	sb.WriteString(dim.Render(strings.Repeat("─", 40)) + "\n")

	// Stdout
	if run.Stdout != "" {
		stdout := run.Stdout
		if markdownMode {
			if renderer, err := glamour.NewTermRenderer(
				glamour.WithStandardStyle("dark"),
				glamour.WithWordWrap(80),
			); err == nil {
				if rendered, err := renderer.Render(run.Stdout); err == nil {
					stdout = rendered
				}
			}
		}
		sb.WriteString(stdout)
		if !strings.HasSuffix(stdout, "\n") {
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString(dim.Render("(no stdout)") + "\n")
	}

	// Stderr section (only if non-empty)
	if run.Stderr != "" {
		sb.WriteString(dim.Render(strings.Repeat("─", 40)) + "\n")
		sb.WriteString(errStyle.Render("stderr:") + "\n")
		sb.WriteString(run.Stderr)
		if !strings.HasSuffix(run.Stderr, "\n") {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// viewInboxDetail renders the inbox run detail as a centered overlay, following
// the same pattern as viewHelpModal.
func (m Model) viewInboxDetail(w, h int, markdownMode bool) string {
	runs := m.inboxModel.Runs()
	if len(runs) == 0 {
		return ""
	}

	idx := m.inboxDetailIdx
	if idx < 0 {
		idx = 0
	}
	if idx >= len(runs) {
		idx = len(runs) - 1
	}
	run := runs[idx]

	mc := m.resolveModalColors()

	innerW := w - 4 // full-width minus border (2) and margin (2)
	if innerW < 40 {
		innerW = 40
	}
	outerW := innerW + 2

	headerStyle := lipgloss.NewStyle().
		Background(mc.titleBG).
		Foreground(mc.titleFG).
		Bold(true).
		Width(innerW).
		Padding(0, 1)

	// Header: "INBOX  [idx+1/total]  kind · name"
	counter := fmt.Sprintf("[%d/%d]", idx+1, len(runs))
	headerText := "INBOX  " +
		lipgloss.NewStyle().Foreground(mc.dim).Render(counter) + "  " +
		lipgloss.NewStyle().Foreground(mc.dim).Render(run.Kind) + " · " +
		lipgloss.NewStyle().Foreground(mc.fg).Render(run.Name)

	// Build scrollable body content.
	content := buildRunContent(run, mc, markdownMode)
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	visibleH := h - 6 // header + border + footer + some padding
	if visibleH < 4 {
		visibleH = 4
	}
	offset := m.inboxDetailScroll
	if offset > len(lines)-visibleH {
		offset = max(len(lines)-visibleH, 0)
	}
	if offset < 0 {
		offset = 0
	}
	end := offset + visibleH
	if end > len(lines) {
		end = len(lines)
	}
	visible := lines[offset:end]

	body := lipgloss.NewStyle().
		Width(innerW).
		Height(visibleH).
		Padding(0, 1).
		Render(strings.Join(visible, "\n"))

	// Footer: scroll hint + key hints.
	total := len(lines)
	dimStyle := lipgloss.NewStyle().Foreground(mc.dim)
	accentStyle := lipgloss.NewStyle().Foreground(mc.accent)
	mdHint := "[m] md"
	if markdownMode {
		mdHint = "[m] raw"
	}
	var footer string
	if total > visibleH {
		scrollHint := accentStyle.Render("j/k  [/]") + dimStyle.Render(" scroll  ")
		keyHints := dimStyle.Render("[n]ext  [p]rev  [q]uit  "+mdHint)
		footer = lipgloss.NewStyle().Foreground(mc.dim).
			Width(innerW).Padding(0, 1).
			Render(scrollHint + keyHints)
	} else {
		footer = lipgloss.NewStyle().Foreground(mc.dim).
			Width(innerW).Padding(0, 1).
			Render("[n]ext  [p]rev  [q]uit  " + mdHint)
	}

	boxContent := strings.Join([]string{
		headerStyle.Render(headerText),
		body,
		footer,
	}, "\n")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(mc.border).
		Width(outerW).
		Render(boxContent)
}
