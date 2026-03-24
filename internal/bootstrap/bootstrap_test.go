package bootstrap_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adam-stokes/orcai/internal/bootstrap"
)

func TestWriteTmuxConf(t *testing.T) {
	dir := t.TempDir()
	confPath, err := bootstrap.WriteTmuxConf(dir, "/fake/orcai")
	if err != nil {
		t.Fatalf("WriteTmuxConf: %v", err)
	}
	data, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("tmux.conf not written: %v", err)
	}
	if len(data) == 0 {
		t.Error("tmux.conf is empty")
	}
	expected := filepath.Join(dir, "tmux.conf")
	if confPath != expected {
		t.Errorf("confPath = %q, want %q", confPath, expected)
	}
	if !strings.Contains(string(data), "status-position bottom") {
		t.Error("tmux.conf missing status-position bottom")
	}
}

func TestSessionExists_NoSuchSession(t *testing.T) {
	if !bootstrap.HasTmux() {
		t.Skip("tmux not in PATH")
	}
	got := bootstrap.SessionExists("orcai-test-nonexistent-xyz")
	if got {
		t.Error("SessionExists returned true for a session that should not exist")
	}
}
