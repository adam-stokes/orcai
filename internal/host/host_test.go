package host_test

import (
	"testing"

	"github.com/adam-stokes/orcai/internal/discovery"
	"github.com/adam-stokes/orcai/internal/host"
)

func TestNewHost(t *testing.T) {
	h := host.New("127.0.0.1:9999")
	if h == nil {
		t.Fatal("expected non-nil host")
	}
}

func TestLoadCLIWrapper(t *testing.T) {
	h := host.New("127.0.0.1:9999")
	p := discovery.Plugin{
		Name:    "shell",
		Command: "bash",
		Type:    discovery.TypeCLIWrapper,
	}
	if err := h.Load(p); err != nil {
		t.Fatalf("Load CLI wrapper: %v", err)
	}
	plugins := h.Plugins()
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}
	if plugins[0].Plugin.Name != "shell" {
		t.Errorf("expected name 'shell', got %q", plugins[0].Plugin.Name)
	}
}
