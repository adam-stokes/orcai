// Package widgetdispatch launches orcai widgets, checking for orcai-<name>
// override binaries in PATH before falling back to `orcai <name>`.
package widgetdispatch

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// Options for dispatch.
type Options struct {
	BusSocket string // passed as --bus-socket if non-empty
}

// Dispatcher is the interface used by layout.Apply.
type Dispatcher interface {
	Dispatch(ctx context.Context, name string, opts Options) error
}

// DefaultDispatcher is the standard PATH-lookup dispatcher.
type DefaultDispatcher struct{}

// Dispatch implements Dispatcher using the package-level Dispatch function.
func (DefaultDispatcher) Dispatch(ctx context.Context, name string, opts Options) error {
	return Dispatch(ctx, name, opts)
}

// Dispatch launches widget `name`, checking for an orcai-<name> override binary
// in PATH before falling back to `orcai <name>`.
func Dispatch(ctx context.Context, name string, opts Options) error {
	bin, args := resolveWidget(name)
	if opts.BusSocket != "" {
		args = append(args, "--bus-socket", opts.BusSocket)
	}

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("widgetdispatch: %s exited: %w", name, err)
	}
	return nil
}

// resolveWidget returns the binary and args to use for the given widget name.
// Checks for orcai-<name> override in PATH, with self-referential detection.
// Falls back to ("orcai", []string{name}).
func resolveWidget(name string) (string, []string) {
	overrideName := "orcai-" + name
	if overridePath, err := exec.LookPath(overrideName); err == nil {
		if !isSelfReferential(overridePath) {
			return overridePath, nil
		}
		log.Printf("widgetdispatch: %s resolves to current orcai binary, using built-in", overrideName)
	}
	// Fall back to orcai <name>
	orcaiBin, err := exec.LookPath("orcai")
	if err != nil {
		// Last resort: use os.Executable
		if self, err2 := os.Executable(); err2 == nil {
			orcaiBin = self
		} else {
			orcaiBin = "orcai"
		}
	}
	return orcaiBin, []string{name}
}

// isSelfReferential returns true if path resolves to the same executable as
// the currently running process.
func isSelfReferential(path string) bool {
	self, err := os.Executable()
	if err != nil {
		return false
	}
	// Resolve symlinks for comparison
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	if resolved, err := filepath.EvalSymlinks(self); err == nil {
		self = resolved
	}
	return path == self
}
