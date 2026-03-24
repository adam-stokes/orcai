package ansiart_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/adam-stokes/orcai/internal/ansiart"
)

func TestLoad_UsesFallback(t *testing.T) {
	got := ansiart.Load("nonexistent.ans", []byte("\x1b[38;5;141mHello\x1b[0m"))
	if got != "\x1b[38;5;141mHello\x1b[0m" {
		t.Errorf("Load() = %q, want fallback content", got)
	}
}

func TestLoad_PrefersUserFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	userPath := filepath.Join(dir, ".config", "orcai", "ui", "test.ans")
	if err := os.MkdirAll(filepath.Dir(userPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(userPath, []byte("custom"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := ansiart.Load("test.ans", []byte("fallback"))
	if got != "custom" {
		t.Errorf("Load() = %q, want %q", got, "custom")
	}
}

func TestLoad_StripsClearScreen(t *testing.T) {
	got := ansiart.Load("x.ans", []byte("before\x1b[2Jafter"))
	if got != "beforeafter" {
		t.Errorf("Load() = %q, want %q", got, "beforeafter")
	}
}

func TestLoad_StripsCursorPosition(t *testing.T) {
	got := ansiart.Load("x.ans", []byte("a\x1b[5;3Hb"))
	if got != "ab" {
		t.Errorf("Load() = %q, want %q", got, "ab")
	}
}

func TestLoad_PreservesColors(t *testing.T) {
	input := "\x1b[38;5;141mPurple\x1b[0m"
	got := ansiart.Load("x.ans", []byte(input))
	if got != input {
		t.Errorf("Load() stripped color codes: got %q", got)
	}
}

func TestClampWidth_ClipsLongLine(t *testing.T) {
	art := "abcdefghij" // 10 visible chars, no ANSI
	got := ansiart.ClampWidth(art, 5)
	if ansiart.VisibleLen(got) != 5 {
		t.Errorf("VisibleLen after ClampWidth(5) = %d, want 5", ansiart.VisibleLen(got))
	}
}

func TestClampWidth_PreservesShortLine(t *testing.T) {
	art := "abc"
	got := ansiart.ClampWidth(art, 10)
	if ansiart.VisibleLen(got) != 3 {
		t.Errorf("short line modified unexpectedly, VisibleLen = %d", ansiart.VisibleLen(got))
	}
}

func TestClampWidth_IgnoresAnsiCodes(t *testing.T) {
	// 3 visible chars wrapped in ANSI — should NOT be clipped at maxCols=5
	art := "\x1b[38;5;141mabc\x1b[0m"
	got := ansiart.ClampWidth(art, 5)
	if ansiart.VisibleLen(got) != 3 {
		t.Errorf("VisibleLen = %d, want 3", ansiart.VisibleLen(got))
	}
}

func TestClampWidth_MultibyteRune(t *testing.T) {
	// "═══" is three box-drawing chars (U+2550, 3 bytes each)
	art := "═══════════" // 11 visible runes
	got := ansiart.ClampWidth(art, 5)
	vl := ansiart.VisibleLen(got)
	if vl != 5 {
		t.Errorf("VisibleLen after ClampWidth(5) on multibyte art = %d, want 5", vl)
	}
}

func TestClampWidth_NoResetOnUnclippedLine(t *testing.T) {
	art := "abc" // 3 chars, limit 10 — not clipped
	got := ansiart.ClampWidth(art, 10)
	if got != "abc" {
		t.Errorf("unclipped line modified: got %q, want %q", got, "abc")
	}
}
