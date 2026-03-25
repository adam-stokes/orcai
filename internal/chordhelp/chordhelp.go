// Package chordhelp runs the global ` chord-key popup.
// Press ` from anywhere in the tmux session to open it.
package chordhelp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/adam-stokes/orcai/internal/bootstrap"
	"github.com/adam-stokes/orcai/internal/styles"
)

type helpState int

const (
	stateHelp helpState = iota
	stateConfirmQuit
	stateConfirmDetach
	stateConfirmReload
)

type model struct {
	state  helpState
	width  int
	height int
	self   string
}

func newModel() model {
	self, _ := os.Executable()
	if resolved, err := filepath.EvalSymlinks(self); err == nil {
		self = resolved
	}
	return model{self: self}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		switch m.state {

		case stateHelp:
			switch msg.String() {
			case "q":
				m.state = stateConfirmQuit
			case "d":
				m.state = stateConfirmDetach
			case "r":
				m.state = stateConfirmReload
			case "n":
				if m.self != "" {
					exec.Command("tmux", "display-popup", "-E",
						"-w", "42", "-h", "20", m.self, "_picker").Run() //nolint:errcheck
				}
				return m, tea.Quit
			case "s":
				if m.self != "" {
					exec.Command("tmux", "display-popup", "-E",
						"-w", "44", "-h", "6", m.self, "_opsx").Run() //nolint:errcheck
				}
				return m, tea.Quit
			case "esc", "`", "ctrl+c":
				return m, tea.Quit
			}

		case stateConfirmQuit, stateConfirmDetach, stateConfirmReload:
			switch msg.String() {
			case "y", "enter":
				switch m.state {
				case stateConfirmQuit:
					exec.Command("tmux", "kill-session", "-t", "orcai").Run() //nolint:errcheck
				case stateConfirmDetach:
					exec.Command("tmux", "detach-client").Run() //nolint:errcheck
				case stateConfirmReload:
					bootstrap.WriteReloadMarker()                          //nolint:errcheck
					exec.Command("tmux", "detach-client").Run()            //nolint:errcheck
				}
				return m, tea.Quit
			case "n", "esc":
				m.state = stateHelp
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	w := m.width
	if w <= 0 {
		w = 44
	}

	headerStyle := lipgloss.NewStyle().
		Background(styles.Purple).
		Foreground(styles.Bg).
		Bold(true).
		Width(w).
		Padding(0, 1)

	// ── Confirmation view ────────────────────────────────────────────────────
	if m.state == stateConfirmQuit || m.state == stateConfirmDetach || m.state == stateConfirmReload {
		var title string
		switch m.state {
		case stateConfirmQuit:
			title = "Quit ORCAI?"
		case stateConfirmDetach:
			title = "Detach session?"
		case stateConfirmReload:
			title = "Reload ORCAI? (sessions preserved)"
		}

		bodyStyle := lipgloss.NewStyle().
			Width(w).
			Padding(1, 2)

		yesStyle := lipgloss.NewStyle().
			Background(styles.Red).
			Foreground(styles.Bg).
			Bold(true).
			Padding(0, 1)

		noStyle := lipgloss.NewStyle().
			Foreground(styles.Comment).
			Padding(0, 1)

		keys := lipgloss.JoinHorizontal(lipgloss.Left,
			yesStyle.Render("y yes"),
			lipgloss.NewStyle().Foreground(styles.Comment).Render("  "),
			noStyle.Render("n no"),
		)

		body := lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Foreground(styles.Red).Bold(true).Render(title),
			"",
			keys,
		)

		return lipgloss.JoinVertical(lipgloss.Left,
			headerStyle.Render("ORCAI  confirm"),
			bodyStyle.Render(body),
		)
	}

	// ── Help view ────────────────────────────────────────────────────────────
	keyStyle := lipgloss.NewStyle().
		Foreground(styles.Pink).
		Bold(true).
		Width(6)

	descStyle := lipgloss.NewStyle().
		Foreground(styles.Fg)

	sectionStyle := lipgloss.NewStyle().
		Foreground(styles.Comment).
		Width(w).
		PaddingTop(1).
		Padding(0, 1)

	rowStyle := lipgloss.NewStyle().
		Width(w).
		Padding(0, 1)

	row := func(key, desc string) string {
		return rowStyle.Render(
			lipgloss.JoinHorizontal(lipgloss.Left,
				keyStyle.Render(key),
				descStyle.Render(desc),
			),
		)
	}

	dimStyle := lipgloss.NewStyle().Foreground(styles.Comment)

	rows := []string{
		headerStyle.Render("ORCAI  shortcuts"),
		sectionStyle.Render("session"),
		row("q", "quit workspace"),
		row("d", "detach  (reconnect with: orcai)"),
		row("r", "reload  (updated binary, sessions preserved)"),
		row("n", "new session"),
		row("s", "OpenSpec — propose a feature"),
		sectionStyle.Render("navigation"),
		row("↑ k", "navigate up"),
		row("↓ j", "navigate down"),
		row("↩", "switch to window"),
		row("x", "kill window"),
		rowStyle.Render(""),
		rowStyle.Render(dimStyle.Render("esc  dismiss")),
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// Run starts the chord-help popup as a bubbletea program.
func Run() {
	p := tea.NewProgram(newModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("chordhelp error: %v\n", err)
	}
}

// RunAction starts the popup directly in a specific state: "quit" or "detach".
// Used when a chord like `q or `d is pressed without going through the help screen.
func RunAction(action string) {
	m := newModel()
	switch action {
	case "quit":
		m.state = stateConfirmQuit
	case "detach":
		m.state = stateConfirmDetach
	case "reload":
		m.state = stateConfirmReload
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("chordhelp error: %v\n", err)
	}
}
