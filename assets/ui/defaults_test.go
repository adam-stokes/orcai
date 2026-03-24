package uiassets

import (
	"regexp"
	"strings"
	"testing"
)

// ansiRe matches any ANSI CSI escape sequence for visible-length measurement.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]`)

// visibleLen returns the number of printable (non-ANSI) runes in s.
func visibleLen(s string) int {
	return len([]rune(ansiRe.ReplaceAllString(s, "")))
}

// innerContent extracts the content between the two ║ border chars on a line.
// Returns ("", false) if the line does not contain ║.
func innerContent(line string) (string, bool) {
	stripped := ansiRe.ReplaceAllString(line, "")
	first := strings.IndexRune(stripped, '║')
	last := strings.LastIndex(stripped, string('║'))
	if first == -1 || first == last {
		return "", false
	}
	return stripped[first+len("║") : last], true
}

func TestWelcomeANSIInnerWidth(t *testing.T) {
	lines := strings.Split(welcomeANSI, "\n")

	const wantWidth = 50

	for i, line := range lines {
		inner, ok := innerContent(line)
		if !ok {
			// Border-only lines (╔...╗, ╚...╝, ╠...╣) — skip
			continue
		}
		got := visibleLen(inner)
		if got != wantWidth {
			t.Errorf("welcomeANSI line %d: inner visible width = %d, want %d\n  raw inner: %q",
				i+1, got, wantWidth, inner)
		}
	}
}

func TestExportedVars(t *testing.T) {
	if len(WelcomeAns) == 0 {
		t.Error("WelcomeAns is empty")
	}
	if string(WelcomeAns) != welcomeANSI {
		t.Error("WelcomeAns content does not match welcomeANSI constant")
	}
}
