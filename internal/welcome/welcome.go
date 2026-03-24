// Package welcome implements the one-time ANSI art splash screen shown in
// window 0 on fresh ORCAI launch. Enter opens the provider picker; any other
// keypress exits to $SHELL via syscall.Exec.
package welcome

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"

	"github.com/adam-stokes/orcai/internal/ansiart"
)

const helpMarkdown = `## Getting Started

Press **` + "`" + `** to open the chord menu from anywhere.

` + "```" + `
` + "`" + `n   new session  (pick AI provider + model)
` + "`" + `q   quit ORCAI
` + "`" + `d   detach       (reconnect later with: orcai)
` + "```" + `

Navigate sessions with **↑ ↓** in the sidebar.
Press **Enter** to start · **x** to kill a session.
`

type model struct {
	userArt  string // non-empty only when ~/.config/orcai/ui/welcome.ans exists
	markdown string
	width    int
	height   int
	self     string
}

func newModel() model {
	self, _ := os.Executable()
	if resolved, err := filepath.EvalSymlinks(self); err == nil {
		self = resolved
	}
	userArt := ansiart.Load("welcome.ans", nil) // "" when no user override
	md, err := glamour.Render(helpMarkdown, "dark")
	if err != nil {
		md = helpMarkdown
	}
	return model{
		userArt:  userArt,
		markdown: md,
		self:     self,
	}
}

// buildWelcomeArt generates the ANSI welcome banner scaled to width columns.
func buildWelcomeArt(width int) string {
	if width < 10 {
		width = 52
	}
	inner := width - 2 // printable columns between the ║ borders

	purple := "\x1b[38;5;141m"
	pink := "\x1b[38;5;212m"
	bold := "\x1b[1;38;5;212m"
	blue := "\x1b[38;5;61m"
	reset := "\x1b[0m"

	pad := func(n int) string {
		if n <= 0 {
			return ""
		}
		return strings.Repeat(" ", n)
	}

	top := purple + "╔" + strings.Repeat("═", inner) + "╗" + reset

	// Visible content: " ░▒▓ O R C A I ▓▒░  Your AI Workspace" = 37 chars
	const logoPrefixLen = 37
	logoLine := purple + "║" + pink + " ░▒▓ " + bold + "O R C A I" + reset +
		pink + " ▓▒░" + blue + "  Your AI Workspace" + pad(inner-logoPrefixLen) +
		purple + "║" + reset

	// Visible content: "      tmux · AI agents · open sessions" = 38 chars
	const subtitlePrefixLen = 38
	subtitleLine := purple + "║" + blue + "      tmux · AI agents · open sessions" +
		pad(inner-subtitlePrefixLen) + purple + "║" + reset

	mid := purple + "╠" + strings.Repeat("═", inner) + "╣" + reset

	scanContent := strings.Repeat("▄▀", inner/2)
	if inner%2 == 1 {
		scanContent += "▄"
	}
	scanLine := purple + "║" + pink + scanContent + purple + "║" + reset

	bot := purple + "╚" + strings.Repeat("═", inner) + "╝" + reset

	return strings.Join([]string{top, logoLine, subtitleLine, mid, scanLine, bot}, "\n")
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		r, err := glamour.NewTermRenderer(
			glamour.WithStylePath("dark"),
			glamour.WithWordWrap(m.width-4),
		)
		if err == nil {
			md, err := r.Render(helpMarkdown)
			if err == nil {
				m.markdown = md
			}
		}
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "enter" && m.self != "" {
			exec.Command("tmux", "display-popup", "-E",
				"-w", "42", "-h", "14", m.self, "_picker").Run() //nolint:errcheck
		}
		return m, tea.Quit
	}
	return m, nil
}

func (m model) View() string {
	w := m.width
	if w <= 0 {
		w = 52
	}
	var art string
	if m.userArt != "" {
		art = ansiart.ClampWidth(m.userArt, w)
	} else {
		art = buildWelcomeArt(w)
	}
	hint := "\x1b[38;5;61m── enter new session · any key continue ──\x1b[0m"
	return strings.Join([]string{art, m.markdown, hint}, "\n")
}

// Run launches the welcome splash TUI. After the user presses any key and
// the program exits, it replaces the current process with $SHELL.
func Run() {
	p := tea.NewProgram(newModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "welcome: %v\n", err)
	}
	execShell()
}

func execShell() {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	if err := syscall.Exec(shell, []string{shell}, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "welcome: exec shell: %v\n", err)
		os.Exit(0)
	}
}
