## Why

Provider definitions for Claude, GitHub Copilot, and Gemini are hardcoded in `internal/picker/picker.go` as a static list, making them impossible to update, configure, or remove without rebuilding orcai. All other providers (opencode, ollama) are already external plugins discovered at runtime. Centralising provider metadata in plugins gives users first-class control, lets plugins declare their own models, and removes the need to fork orcai to add or change a provider.

## What Changes

- Add three new orcai CLI-wrapper plugins to the `orcai-plugins` repo: `claude`, `github-copilot`, `gemini`
- Each plugin is a Go binary (`main.go`) that wraps the corresponding upstream CLI tool
- Each plugin ships a companion `<name>.yaml` sidecar descriptor that declares the plugin ID, command, launch args, and available models
- **BREAKING**: Remove `claude`, `copilot`, and `gemini` (once added) from the embedded `Providers` slice in `internal/picker/picker.go`; the static list retains only `ollama` and `shell` as built-ins
- The `buildProviders()` discovery loop already handles `TypeCLIWrapper` extras (fixed in a prior commit); no changes to discovery logic are needed
- The switchboard agent runner reads models from the discovered `ProviderDef.Models` list, which is now populated from the sidecar YAML

## Capabilities

### New Capabilities
- `claude-plugin`: Standalone orcai CLI-wrapper plugin for Claude — wraps `claude --print`, ships sidecar YAML declaring opus-4-6/sonnet-4-6/haiku-4-5 models
- `copilot-plugin`: Standalone orcai CLI-wrapper plugin for GitHub Copilot — wraps `gh copilot suggest` or `gh copilot explain`, ships sidecar YAML
- `gemini-plugin`: Standalone orcai CLI-wrapper plugin for Gemini — wraps `gemini` CLI, ships sidecar YAML declaring gemini-2.0-flash / gemini-1.5-pro models

### Modified Capabilities
- `cli-adapter-sidecar`: Sidecar YAML schema must support a `models` list field so plugins can declare available models that orcai surfaces in the agent runner without hardcoding them in picker.go

## Impact

- `orcai-plugins` repo: new `plugins/claude/`, `plugins/github-copilot/`, `plugins/gemini/` directories
- `internal/picker/picker.go`: remove claude/copilot from static `Providers`; update `pipelineLaunchArgs` to remove hardcoded claude entry (it moves to the sidecar)
- `internal/plugin/wrapper.go` (or equivalent): extend wrapper loader to read `models` from sidecar YAML and populate `ProviderDef.Models`
- `internal/picker/picker.go`: `BuildProviders` / `buildProviders` must populate models from loaded sidecar when `TypeCLIWrapper` extras are appended
- No changes to pipeline runner, switchboard, or bootstrap
