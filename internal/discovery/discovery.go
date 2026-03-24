package discovery

import (
	"os"
	"os/exec"
	"path/filepath"
)

// PluginType distinguishes native gRPC plugins from auto-detected CLI wrappers.
type PluginType int

const (
	TypeNative     PluginType = iota // implements OrcaiPlugin gRPC service
	TypeCLIWrapper                   // auto-detected tool in PATH
)

// Plugin describes a discovered plugin or CLI wrapper.
type Plugin struct {
	Name    string
	Command string
	Args    []string
	Type    PluginType
}

// knownCLITools is the built-in registry of AI CLI tools orcai auto-detects.
var knownCLITools = []Plugin{
	{Name: "claude",   Command: "claude"},
	{Name: "opencode", Command: "opencode"},
	{Name: "copilot",  Command: "gh", Args: []string{"copilot"}},
	{Name: "gemini",   Command: "gemini"},
	{Name: "aider",    Command: "aider"},
	{Name: "goose",    Command: "goose"},
}

// Discover returns all available plugins: Tier 1 (native, from pluginsDir) and
// Tier 2 (CLI wrappers from PATH). Native plugins shadow CLI wrappers of the same name.
func Discover(pluginsDir string) ([]Plugin, error) {
	native, err := scanNative(pluginsDir)
	if err != nil {
		return nil, err
	}

	nativeNames := make(map[string]bool, len(native))
	for _, p := range native {
		nativeNames[p.Name] = true
	}

	plugins := native
	for _, tool := range knownCLITools {
		if nativeNames[tool.Name] {
			continue // native plugin takes priority
		}
		if _, err := exec.LookPath(tool.Command); err == nil {
			t := tool
			t.Type = TypeCLIWrapper
			plugins = append(plugins, t)
		}
	}
	return plugins, nil
}

func scanNative(dir string) ([]Plugin, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var plugins []Plugin
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil || info.Mode()&0o111 == 0 {
			continue // not executable
		}
		plugins = append(plugins, Plugin{
			Name:    e.Name(),
			Command: filepath.Join(dir, e.Name()),
			Type:    TypeNative,
		})
	}
	return plugins, nil
}
