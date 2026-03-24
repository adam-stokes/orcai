package container

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestTrustCopilot(t *testing.T) {
	// Use a temp dir as home
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	copilotDir := filepath.Join(tmp, ".copilot")
	if err := os.MkdirAll(copilotDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(copilotDir, "config.json")

	// Pre-populate with an existing field so we can verify it is preserved
	initial := map[string]interface{}{
		"banner":          "never",
		"trusted_folders": []string{"/existing/path"},
	}
	data, _ := json.Marshal(initial)
	os.WriteFile(cfgPath, data, 0o644)

	trustCopilot("/new/path")

	out, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("config not written: %v", err)
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(out, &cfg); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// trusted_folders must contain both paths
	raw, ok := cfg["trusted_folders"].([]interface{})
	if !ok {
		t.Fatal("trusted_folders missing or wrong type")
	}
	found := false
	for _, v := range raw {
		if v.(string) == "/new/path" {
			found = true
		}
	}
	if !found {
		t.Error("new path not added to trusted_folders")
	}

	// existing field must be preserved
	if cfg["banner"] != "never" {
		t.Error("other fields not preserved")
	}

	// idempotent: calling again should not duplicate
	trustCopilot("/new/path")
	out2, _ := os.ReadFile(cfgPath)
	var cfg2 map[string]interface{}
	json.Unmarshal(out2, &cfg2)
	raw2 := cfg2["trusted_folders"].([]interface{})
	count := 0
	for _, v := range raw2 {
		if v.(string) == "/new/path" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 occurrence of /new/path, got %d", count)
	}
}
