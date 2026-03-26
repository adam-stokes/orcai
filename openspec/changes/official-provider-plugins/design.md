## Context

Three AI providers — Claude, GitHub Copilot, and Gemini — are currently defined as static entries in `internal/picker/picker.go`. Ollama and opencode are already external plugins: Go binaries wrapped by sidecar YAML files in `~/.config/orcai/wrappers/`. The switchboard agent runner builds its provider list by calling `picker.BuildProviders()`, which calls `buildProviders()`. That function discovers plugins via `discovery.Discover`, marks them as `TypeCLIWrapper` or `TypeNative`, then appends any extras not in the static list. The static list is the only place where models are declared today (e.g. `claude-opus-4-6`, `claude-sonnet-4-6`).

The `SidecarSchema` struct (`internal/plugin/cli_adapter.go`) currently has no `models` field. The discovery layer returns `PluginInfo` objects with no model metadata. The picker's `buildProviders` function therefore has no way to populate `ProviderDef.Models` from discovered plugins — it can only inject ollama models via a special runtime query, and opencode gets ollama models via a hardcoded name check.

## Goals / Non-Goals

**Goals:**
- Three new plugins in `orcai-plugins/plugins/`: `claude/`, `github-copilot/`, `gemini/`
- Each plugin ships a sidecar YAML that declares its `models` list
- Extend `SidecarSchema` with a `models []SidecarModel` field (id + label)
- `buildProviders` populates `ProviderDef.Models` from the sidecar when appending `TypeCLIWrapper` extras
- Remove `claude` and `copilot` from the embedded `Providers` slice in picker.go
- Remove the hardcoded claude entry from `pipelineLaunchArgs` (it moves to sidecar `args`)
- Remove the hardcoded `opencode` name check for ollama model injection (opencode sidecar declares its own models)

**Non-Goals:**
- Dynamic model discovery at runtime for Gemini or Copilot (hardcoded in sidecar is fine)
- Changes to the pipeline runner or switchboard view logic
- Changes to `discovery.Discover` or `PluginInfo` struct

## Decisions

### D1: Extend SidecarSchema with a `models` field

Add to `SidecarSchema`:
```go
type SidecarModel struct {
    ID    string `yaml:"id"`
    Label string `yaml:"label"`
}
// in SidecarSchema:
Models []SidecarModel `yaml:"models"`
```

**Why**: The sidecar is already the authoritative descriptor for a CLI wrapper plugin. Adding models here keeps all provider metadata in one place (the YAML file), eliminates picker.go hardcoding, and makes it trivial to add/remove models without recompiling orcai.

**Alternative considered**: a separate `<name>-models.yaml` file. Rejected — unnecessary indirection.

### D2: Populate ProviderDef.Models in buildProviders from sidecar

`buildProviders` currently appends `ProviderDef{ID: name, Label: name}` for discovered extras. The `PluginInfo` returned by `discovery.Discover` already has a `Path` field (the sidecar path for `TypeCLIWrapper`). We re-parse the sidecar YAML at provider-build time to extract `models`.

**Why**: Avoids threading model metadata through the discovery layer (`PluginInfo`). The sidecar file is cheap to re-read (called once at startup). Keeps discovery layer simple.

**Alternative**: Add `Models` to `PluginInfo`. Rejected — discovery is not provider-specific; not all plugins are AI providers.

### D3: Sidecar YAML location for new plugins

New plugins place their sidecar YAML at `plugins/<name>/<name>.yaml` in the `orcai-plugins` repo and document that users should copy it to `~/.config/orcai/wrappers/<name>.yaml`. Installation via `make install` (same as ollama/opencode).

### D4: Plugin binaries for claude, copilot, gemini

Each binary reads prompt from stdin, passes `--model` from `ORCAI_MODEL` env or `--model` arg, and writes response to stdout. Claude uses `claude --print`; Copilot uses `gh copilot suggest -t shell`; Gemini uses `gemini` CLI with model flag. The binary handles the non-interactive invocation that the raw CLI may not do cleanly on its own.

### D5: Remove hardcoded `opencode` model injection

The current `buildProviders` has `if name == "opencode" { p = injectOllamaModels(...) }`. After this change, opencode's sidecar should declare its models directly. We remove the special-case code and update the opencode sidecar YAML to list the ollama models.

## Risks / Trade-offs

- [Risk] Users with existing orcai installations lose claude/copilot in the agent runner until they install the new plugin sidecars → Mitigation: keep `ollama` and `shell` in static list; print a warning on first run if no providers are discovered
- [Risk] `gemini` CLI flags may differ across versions → Mitigation: plugin binary can normalise flags; easy to update the binary without changing orcai
- [Risk] Re-reading sidecar YAML in `buildProviders` adds a small I/O cost → Mitigation: negligible; only runs at startup and on `r` refresh

## Migration Plan

1. Extend `SidecarSchema` + `buildProviders` in orcai (orcai repo)
2. Add new plugin directories and binaries in orcai-plugins repo
3. Update opencode sidecar YAML to declare models (removes hardcoded injection)
4. Remove claude/copilot from static `Providers` and `pipelineLaunchArgs` in picker.go
5. Update `~/.config/orcai/wrappers/` installation docs

No database migrations or API changes. Rollback: restore static Providers list.
