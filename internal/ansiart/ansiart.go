// Package ansiart loads and sanitises ANSI art files for use inside bubbletea.
// It checks ~/.config/orcai/ui/<name> for user overrides before falling back
// to embedded defaults.
package ansiart

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

// stripRe removes ANSI sequences that would corrupt bubbletea's renderer:
// clear-screen (\x1b[2J), absolute cursor positioning (\x1b[n;mH or \x1b[H),
// and cursor show/hide (\x1b[?25l / \x1b[?25h).
var stripRe = regexp.MustCompile(`\x1b\[(?:2J|[0-9;]*H|\?25[lh])`)

// ansiRe matches any ANSI CSI escape sequence (used for visible-length counting).
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]`)

// Load returns the ANSI art for name (e.g. "welcome.ans").
// User file at ~/.config/orcai/ui/<name> takes precedence over embedded fallback.
// Sequences that would corrupt bubbletea rendering are stripped.
func Load(name string, embedded []byte) string {
	home, err := os.UserHomeDir()
	if err == nil {
		userPath := filepath.Join(home, ".config", "orcai", "ui", name)
		if data, err := os.ReadFile(userPath); err == nil {
			return strip(string(data))
		}
	}
	return strip(string(embedded))
}

// ClampWidth clips each visible line of ANSI art to maxCols printable characters.
// ANSI escape sequences within the kept portion are preserved; a reset is appended
// to each clipped line to prevent colour bleed.
func ClampWidth(art string, maxCols int) string {
	if maxCols <= 0 {
		return art
	}
	lines := strings.Split(art, "\n")
	for i, line := range lines {
		lines[i] = clampLine(line, maxCols)
	}
	return strings.Join(lines, "\n")
}

// VisibleLen returns the number of printable (non-ANSI) runes in s.
// Exported so callers can measure art height/width for layout decisions.
func VisibleLen(s string) int {
	return len([]rune(ansiRe.ReplaceAllString(s, "")))
}

func strip(s string) string {
	return stripRe.ReplaceAllString(s, "")
}

func clampLine(line string, maxCols int) string {
	visible := 0
	clipped := false
	var out strings.Builder
	i := 0
	for i < len(line) {
		// Consume escape sequences without counting them as visible.
		if line[i] == '\x1b' {
			if i+1 < len(line) && line[i+1] == '[' {
				// CSI sequence: ESC [ ... <final-letter>
				j := i + 2
				for j < len(line) && (line[j] == ';' || line[j] == '?' ||
					(line[j] >= '0' && line[j] <= '9')) {
					j++
				}
				if j < len(line) {
					j++ // consume final letter
				}
				out.WriteString(line[i:j])
				i = j
			} else if i+1 < len(line) {
				// Non-CSI two-byte ESC sequence (e.g. ESC M) — pass through, not visible
				out.WriteString(line[i : i+2])
				i += 2
			} else {
				// Bare ESC at end of line — pass through
				out.WriteByte('\x1b')
				i++
			}
			continue
		}
		if visible >= maxCols {
			clipped = true
			break
		}
		_, size := utf8.DecodeRuneInString(line[i:])
		out.WriteString(line[i : i+size])
		visible++
		i += size
	}
	if clipped {
		out.WriteString("\x1b[0m") // reset prevents colour bleed
	}
	return out.String()
}
