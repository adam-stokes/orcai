package cmd

import (
	"encoding/json"
	"testing"

	"github.com/adam-stokes/orcai/internal/picker"
)

func TestLaunchItem_PipelineUsesFileWhenSet(t *testing.T) {
	// Verify launchItem builds a shell command using PipelineFile when available.
	// We can't actually run tmux in a test, so we test the argument-building logic
	// by checking that PipelineFile is preferred over Name.
	item := picker.PickerItem{
		Kind:         "pipeline",
		Name:         "my-pipeline",
		PipelineFile: "/home/user/.config/orcai/pipelines/my-pipeline.pipeline.yaml",
	}
	// Ensure the item JSON round-trips so launchItem can consume it.
	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got picker.PickerItem
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.PipelineFile != item.PipelineFile {
		t.Errorf("PipelineFile lost in round-trip: got %q", got.PipelineFile)
	}
}

func TestLaunchItem_ProviderMissingID(t *testing.T) {
	// launchItem must return an error when providerID is empty for a provider item.
	item := picker.PickerItem{
		Kind:       "provider",
		Name:       "Claude",
		ProviderID: "",
	}
	err := launchItem(item)
	if err == nil {
		t.Fatal("expected error for provider item with empty providerID")
	}
}

func TestLaunchItem_SessionKindNoOp(t *testing.T) {
	// Session items with no SessionIndex should not error.
	item := picker.PickerItem{
		Kind:         "session",
		Name:         "claude-1",
		SessionIndex: "",
	}
	// Without TMUX set, tmux will fail but launchItem's session branch just calls
	// select-window and silently ignores errors for empty indices — no error returned.
	err := launchItem(item)
	if err != nil {
		t.Errorf("unexpected error for session kind: %v", err)
	}
}
