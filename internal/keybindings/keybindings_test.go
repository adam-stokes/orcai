package keybindings

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadConfig_FileAbsent verifies that a missing file returns an empty
// config and nil error.
func TestLoadConfig_FileAbsent(t *testing.T) {
	cfg, err := LoadConfig("/tmp/orcai-keybindings-does-not-exist-xyz.yaml")
	if err != nil {
		t.Fatalf("LoadConfig absent: unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadConfig absent: expected non-nil config")
	}
	if len(cfg.Bindings) != 0 {
		t.Errorf("LoadConfig absent: expected 0 bindings, got %d", len(cfg.Bindings))
	}
}

// TestLoadConfig_Valid verifies that a well-formed keybindings.yaml is parsed.
func TestLoadConfig_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keybindings.yaml")
	content := `
bindings:
  - key: "M-n"
    action: launch-session-picker
  - key: "M-t"
    action: open-sysop
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig valid: %v", err)
	}
	if len(cfg.Bindings) != 2 {
		t.Fatalf("expected 2 bindings, got %d", len(cfg.Bindings))
	}
	if cfg.Bindings[0].Key != "M-n" {
		t.Errorf("bindings[0].Key = %q; want M-n", cfg.Bindings[0].Key)
	}
	if cfg.Bindings[1].Action != "open-sysop" {
		t.Errorf("bindings[1].Action = %q; want open-sysop", cfg.Bindings[1].Action)
	}
}

// TestApply_EmptyConfig verifies that Apply with an empty config returns nil.
func TestApply_EmptyConfig(t *testing.T) {
	cfg := &Config{}
	err := Apply(cfg)
	if err != nil {
		t.Fatalf("Apply empty config: unexpected error: %v", err)
	}
}

// TestApply_UnknownAction verifies that Apply skips unknown actions and does
// not return an error.
func TestApply_UnknownAction(t *testing.T) {
	cfg := &Config{
		Bindings: []Binding{
			{Key: "M-x", Action: "nonexistent-action-xyz"},
		},
	}
	err := Apply(cfg)
	if err != nil {
		t.Fatalf("Apply unknown action: unexpected error: %v", err)
	}
}
