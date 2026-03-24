package container_test

import (
	"testing"

	"github.com/adam-stokes/orcai/container"
)

func TestDockerRuntimeName(t *testing.T) {
	r := &container.DockerRuntime{}
	if r.Name() != "docker" {
		t.Fatalf("expected docker, got %s", r.Name())
	}
}

func TestDetectReturnsRuntime(t *testing.T) {
	r := container.Detect()
	if r == nil {
		t.Fatal("Detect() returned nil")
	}
	name := r.Name()
	if name != "docker" && name != "apple" {
		t.Fatalf("unexpected runtime name: %s", name)
	}
}

func TestContainerName(t *testing.T) {
	name := container.ContainerName("abc123")
	if name != "stok-session-abc123" {
		t.Fatalf("expected stok-session-abc123, got %s", name)
	}
}
