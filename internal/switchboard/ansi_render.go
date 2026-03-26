package switchboard

import (
	"bytes"
	"strings"

	"github.com/adam-stokes/orcai/internal/themes"
)

// panelTitles maps panel keys to their plain-text fallback titles.
var panelTitles = map[string]string{
	"pipelines":     "PIPELINES",
	"agent_runner":  "AGENT RUNNER",
	"signal_board":  "SIGNAL BOARD",
	"activity_feed": "ACTIVITY FEED",
}

// spriteWidth returns the visual width of the widest line in ans bytes,
// ignoring ANSI escape sequences.
func spriteWidth(ans []byte) int {
	maxW := 0
	for _, line := range bytes.Split(ans, []byte("\n")) {
		w := visibleWidth(string(line))
		if w > maxW {
			maxW = w
		}
	}
	return maxW
}

// visibleWidth returns the printable character count of s, stripping ANSI escapes.
func visibleWidth(s string) int {
	inEsc := false
	w := 0
	i := 0
	for i < len(s) {
		b := s[i]
		if b == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			inEsc = true
			i += 2
			continue
		}
		if inEsc {
			if (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') {
				inEsc = false
			}
			i++
			continue
		}
		// Decode UTF-8 rune and count it as one visible column.
		_, size := decodeRuneAt(s, i)
		w++
		i += size
	}
	return w
}

// decodeRuneAt decodes the UTF-8 rune starting at s[i], returning the rune
// and its byte length.
func decodeRuneAt(s string, i int) (rune, int) {
	runes := []rune(s[i:minInt(i+4, len(s))])
	if len(runes) == 0 {
		return 0, 1
	}
	return runes[0], len(string(runes[0]))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// RenderHeader returns the plain-text fallback title for a panel.
// Used in boxTop() when no ANS sprite is available.
func RenderHeader(panel string) string {
	if title, ok := panelTitles[panel]; ok {
		return title
	}
	return strings.ToUpper(panel)
}

// SpriteLines returns the ANS sprite for a panel as individual lines, ready
// to be prepended in place of a boxTop() call.
//
// Returns nil when:
//   - the bundle has no sprite for this panel, or
//   - panelWidth > 0 and the widest sprite line exceeds panelWidth.
//
// The last returned line has "\x1b[0m" appended to prevent color bleed into
// subsequent box rows.
func SpriteLines(bundle *themes.Bundle, panel string, panelWidth int) []string {
	if bundle == nil || bundle.HeaderBytes == nil {
		return nil
	}
	ans, ok := bundle.HeaderBytes[panel]
	if !ok || len(ans) == 0 {
		return nil
	}
	// Enforce width constraint.
	if panelWidth > 0 && spriteWidth(ans) > panelWidth {
		return nil
	}
	// Split into non-empty lines.
	var lines []string
	for _, raw := range bytes.Split(ans, []byte("\n")) {
		s := strings.TrimRight(string(raw), "\r")
		if visibleWidth(s) > 0 {
			lines = append(lines, s)
		}
	}
	if len(lines) == 0 {
		return nil
	}
	lines[len(lines)-1] += "\x1b[0m"
	return lines
}
