package container_test

import "testing"
import "github.com/adam-stokes/orcai/container"

func TestAppleRuntimeName(t *testing.T) {
	r := &container.AppleRuntime{}
	if r.Name() != "apple" {
		t.Fatalf("expected apple, got %s", r.Name())
	}
}

func TestAppleRuntimeAvailable(t *testing.T) {
	r := &container.AppleRuntime{}
	// Available() should not panic regardless of whether 'container' CLI is installed.
	_ = r.Available()
}
