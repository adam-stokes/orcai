## Why

The New Session picker hard-codes a static list of AI providers (Claude, OpenCode, GitHub Copilot, Ollama, Shell) that is disconnected from the plugin system now powering orcai's runtime. Pipelines are listed in the picker but select them doesn't launch an actual shell session — the `orcai new` command is stubbed with a TODO. Both issues leave the picker non-functional for the two most important session types.

## What Changes

- **BREAKING** Remove the static `— providers —` section from the picker; replace with dynamic provider entries derived from installed plugins (plugins with `category: providers` or matching core provider IDs whose binary is in PATH)
- Implement `orcai new` (currently a TODO stub) to handle all three launch modes: provider session, pipeline session, plain shell
- When a pipeline is selected, launch a new tmux window running `orcai pipeline run <pipeline-file>` (an interactive shell session with the pipeline executing), not a bare shell
- Providers only appear in the picker when their backing plugin is registered in the plugin Manager OR their binary exists in PATH (matching current bundled profile `Binary` field)
- Core provider IDs (`claude`, `opencode`, `copilot`, `ollama`) are shown only when the corresponding plugin/binary is detected; no ghost entries

## Capabilities

### New Capabilities

- `session-launcher`: The `orcai new` command — receives a serialised `PickerItem` from the picker popup and launches the correct tmux window (provider session, pipeline shell, or bare shell)

### Modified Capabilities

- `welcome-dashboard`: Picker provider section must be replaced — providers now come from plugin discovery, not bundled static list; pipeline items must carry the pipeline file path so the launcher can use it

## Impact

- `internal/picker/picker.go` — `BuildProviders()` replaced by plugin-aware discovery function
- `cmd/new.go` — stub TODO replaced with full launch logic
- `internal/providers/registry.go` — may still be used for profile metadata (window name, launch args) but no longer the source of truth for availability
- `internal/plugin/manager.go` / `internal/plugin/discovery.go` — read at picker startup to enumerate available plugins
- No changes to pipeline execution engine, plugin Manager internals, or tmux session structure
