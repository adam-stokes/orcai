## ADDED Requirements

### Requirement: Sidecar YAML models field
`SidecarSchema` SHALL include a `Models []SidecarModel` field where `SidecarModel` has `ID string` and `Label string`. When a sidecar file declares a `models` list, `NewCliAdapterFromSidecar` SHALL capture it in the returned adapter so callers can retrieve it.

#### Scenario: Sidecar with models list parsed correctly
- **WHEN** a sidecar file contains a `models:` block with two entries `[{id: foo, label: Foo}, {id: bar, label: Bar}]`
- **THEN** the loaded adapter exposes both models via `Models()` (or equivalent accessor)

#### Scenario: Sidecar without models field returns empty list
- **WHEN** a sidecar file omits the `models:` field entirely
- **THEN** the loaded adapter returns an empty (not nil) models list

### Requirement: buildProviders reads models from sidecar for CLI wrapper extras
`buildProviders` SHALL, when appending a discovered `TypeCLIWrapper` plugin that is not in the static provider list, re-read the sidecar YAML and populate `ProviderDef.Models` from the sidecar's `models` field.

#### Scenario: Discovered plugin with sidecar models appears with models in agent runner
- **WHEN** a sidecar at `~/.config/orcai/wrappers/claude.yaml` declares three models
- **THEN** `buildProviders` returns a `ProviderDef` for `claude` with all three models populated

#### Scenario: Discovered plugin with no models in sidecar has empty model list
- **WHEN** a sidecar omits the `models` field
- **THEN** `buildProviders` returns a `ProviderDef` with an empty `Models` slice, and the agent runner skips the model selection step

## REMOVED Requirements

### Requirement: Static Providers list in picker.go
**Reason**: All AI provider definitions move to external sidecar YAML files. The static list was the only source of model metadata for claude and copilot; that role is replaced by sidecar-declared models. Hardcoding providers in orcai prevents users from customising or removing providers without recompiling.
**Migration**: Install the corresponding `~/.config/orcai/wrappers/<provider>.yaml` sidecar file. The `ollama` and `shell` built-in entries are retained as they have no external plugin equivalents.
