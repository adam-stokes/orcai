// Package welcome implements the ABS welcome dashboard BubbleTea widget.
//
// It is used by the `orcai welcome` cobra subcommand. Unlike the standalone
// orcai-welcome binary, this version exits normally when the user dismisses the
// dashboard — it does NOT exec a shell replacement.
package welcome

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/adam-stokes/orcai/internal/busd"
)

// ── Bus protocol ───────────────────────────────────────────────────────────────

const (
	busDialTimeout = 2 * time.Second
	busWidgetName  = "orcai-welcome"
)

var busSubscriptions = []string{
	"theme.changed",
	"session.started",
	"session.ended",
	"orcai.telemetry",
}

// registrationFrame is sent to the bus daemon on connect.
type registrationFrame struct {
	Name      string   `json:"name"`
	Subscribe []string `json:"subscribe"`
}

// busEvent is a decoded server-to-client frame.
type busEvent struct {
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

// themeChangedPayload is the payload for the "theme.changed" event.
type themeChangedPayload struct {
	Name string `json:"name"`
}

// ── Tea messages ───────────────────────────────────────────────────────────────

// themeChangedMsg is sent to the BubbleTea program when the active theme changes.
type themeChangedMsg struct {
	ThemeName string
}

// ── ANSI palette ───────────────────────────────────────────────────────────────

// palette holds the ANSI escape sequences used to render the welcome screen.
type palette struct {
	purple string
	pink   string
	bold   string
	blue   string
	dim    string
	reset  string
}

// absDefaults returns the ABS/Dracula default palette.
func absDefaults() palette {
	return palette{
		purple: "\x1b[38;5;141m",
		pink:   "\x1b[38;5;212m",
		bold:   "\x1b[1;38;5;212m",
		blue:   "\x1b[38;5;61m",
		dim:    "\x1b[38;5;66m",
		reset:  "\x1b[0m",
	}
}

// paletteForTheme returns a palette for the given theme name.
func paletteForTheme(name string) palette {
	switch name {
	case "abs", "":
		return absDefaults()
	default:
		return absDefaults()
	}
}

// ── Banner / Help ───────────────────────────────────────────────────────────────

func buildWelcomeArt(width int, p palette) string {
	if width < 10 {
		width = 52
	}
	inner := width - 2

	pad := func(n int) string {
		if n <= 0 {
			return ""
		}
		return strings.Repeat(" ", n)
	}

	top := p.purple + "╔" + strings.Repeat("═", inner) + "╗" + p.reset

	const logoPrefixLen = 37
	logoLine := p.purple + "║" + p.pink + " ░▒▓ " + p.bold + "O R C A I" + p.reset +
		p.pink + " ▓▒░" + p.blue + "  Your AI Workspace" + pad(inner-logoPrefixLen) +
		p.purple + "║" + p.reset

	const subtitlePrefixLen = 38
	subtitleLine := p.purple + "║" + p.blue + "      tmux · AI agents · open sessions" +
		pad(inner-subtitlePrefixLen) + p.purple + "║" + p.reset

	mid := p.purple + "╠" + strings.Repeat("═", inner) + "╣" + p.reset

	scanContent := strings.Repeat("▄▀", inner/2)
	if inner%2 == 1 {
		scanContent += "▄"
	}
	scanLine := p.purple + "║" + p.pink + scanContent + p.purple + "║" + p.reset

	bot := p.purple + "╚" + strings.Repeat("═", inner) + "╝" + p.reset

	return strings.Join([]string{top, logoLine, subtitleLine, mid, scanLine, bot}, "\n")
}

func buildHelp(width int, p palette) string {
	col := p.dim + strings.Repeat("─", width) + p.reset

	lines := []string{
		col,
		"",
		p.blue + "  Press  " + p.pink + "ctrl+space" + p.blue + "  to open the chord menu from anywhere." + p.reset,
		"",
		p.blue + "    " + p.pink + "n" + p.dim + "  new session   " + p.blue + "(pick AI provider + model)" + p.reset,
		p.blue + "    " + p.pink + "t" + p.dim + "  sysop panel   " + p.blue + "(agent monitor in current window)" + p.reset,
		p.blue + "    " + p.pink + "p" + p.dim + "  prompt builder" + p.blue + p.reset,
		p.blue + "    " + p.pink + "q" + p.dim + "  quit ORCAI" + p.reset,
		p.blue + "    " + p.pink + "d" + p.dim + "  detach        " + p.blue + "(reconnect later: orcai)" + p.reset,
		"",
		col,
		"",
		p.dim + "  ── enter new session · any key continue ──" + p.reset,
	}
	return strings.Join(lines, "\n")
}

// ── BubbleTea model ────────────────────────────────────────────────────────────

type model struct {
	width        int
	height       int
	self         string
	palette      palette
	launchPicker bool
}

// resolvePickerBin returns the path to the orcai-picker binary.
func resolvePickerBin() string {
	if bin, err := exec.LookPath("orcai-picker"); err == nil {
		return bin
	}
	self, _ := os.Executable()
	if resolved, err := filepath.EvalSymlinks(self); err == nil {
		self = resolved
	}
	return filepath.Join(filepath.Dir(self), "orcai-picker")
}

func newModel() model {
	return model{
		self:    resolvePickerBin(),
		palette: absDefaults(),
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case themeChangedMsg:
		m.palette = paletteForTheme(msg.ThemeName)
	case tea.KeyMsg:
		if msg.String() == "enter" && m.self != "" {
			m.launchPicker = true
		}
		return m, tea.Quit
	}
	return m, nil
}

func (m model) View() string {
	w := m.width
	if w <= 0 {
		w = 80
	}
	return buildWelcomeArt(w, m.palette) + "\n" + buildHelp(w, m.palette)
}

// ── Bus connection ─────────────────────────────────────────────────────────────

// connectBus dials the busd socket (using sockPath if non-empty, otherwise
// auto-discovering via busd.SocketPath). Returns nil if the daemon is not
// running — this is non-fatal.
func connectBus(sockPath string) net.Conn {
	if sockPath == "" {
		var err error
		sockPath, err = busd.SocketPath()
		if err != nil {
			return nil
		}
	}

	conn, err := net.DialTimeout("unix", sockPath, busDialTimeout)
	if err != nil {
		return nil
	}

	reg := registrationFrame{
		Name:      busWidgetName,
		Subscribe: busSubscriptions,
	}
	data, _ := json.Marshal(reg)
	data = append(data, '\n')
	conn.SetWriteDeadline(time.Now().Add(busDialTimeout)) //nolint:errcheck
	conn.Write(data)                                      //nolint:errcheck
	conn.SetWriteDeadline(time.Time{})                    //nolint:errcheck

	return conn
}

// readBusEvents reads newline-delimited JSON frames from conn and forwards
// relevant messages to the BubbleTea program p. It runs until conn is closed.
func readBusEvents(conn net.Conn, p *tea.Program) {
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Bytes()
		var ev busEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			continue
		}
		switch ev.Event {
		case "theme.changed":
			var pl themeChangedPayload
			if err := json.Unmarshal(ev.Payload, &pl); err == nil {
				p.Send(themeChangedMsg{ThemeName: pl.Name})
			}
		}
	}
}

// ── Entry point ────────────────────────────────────────────────────────────────

// Run starts the welcome dashboard BubbleTea program. When busSocket is
// non-empty it is used as the socket path; otherwise auto-discovery via
// busd.SocketPath() is attempted. Unlike the standalone orcai-welcome binary,
// this function exits normally when the user dismisses the dashboard — it does
// NOT exec a shell replacement.
func Run(busSocket string) error {
	conn := connectBus(busSocket)

	p := tea.NewProgram(newModel(), tea.WithAltScreen())

	if conn != nil {
		go readBusEvents(conn, p)
		defer conn.Close()
	}

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("welcome: %w", err)
	}

	if m, ok := finalModel.(model); ok && m.launchPicker && m.self != "" {
		exec.Command("tmux", "display-popup", "-E", "-w", "120", "-h", "40", m.self).Run() //nolint:errcheck
	}

	return nil
}
