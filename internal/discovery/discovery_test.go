package discovery_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/adam-stokes/orcai/internal/discovery"
)

func TestScanNative_Empty(t *testing.T) {
	dir := t.TempDir()
	plugins, err := discovery.Discover(dir)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	for _, p := range plugins {
		if p.Type == discovery.TypeNative {
			t.Errorf("expected no native plugins in empty dir, got %+v", p)
		}
	}
}

func TestScanNative_FindsExecutable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orcai-test-plugin")
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho hi"), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	plugins, err := discovery.Discover(dir)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	found := false
	for _, p := range plugins {
		if p.Name == "orcai-test-plugin" && p.Type == discovery.TypeNative {
			found = true
		}
	}
	if !found {
		t.Errorf("expected to find orcai-test-plugin, got %+v", plugins)
	}
}

func TestScanNative_SkipsNonExecutable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "not-executable")
	if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	plugins, err := discovery.Discover(dir)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	for _, p := range plugins {
		if p.Name == "not-executable" {
			t.Errorf("should not have loaded non-executable file")
		}
	}
}

func TestNativePriorityOverCLI(t *testing.T) {
	dir := t.TempDir()
	// Create a native plugin named "claude" — it should shadow the CLI wrapper
	path := filepath.Join(dir, "claude")
	if err := os.WriteFile(path, []byte("#!/bin/sh"), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	plugins, err := discovery.Discover(dir)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	count := 0
	for _, p := range plugins {
		if p.Name == "claude" {
			count++
			if p.Type != discovery.TypeNative {
				t.Errorf("expected claude to be TypeNative, got %v", p.Type)
			}
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 claude plugin, got %d", count)
	}
}
