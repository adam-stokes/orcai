## ADDED Requirements

### Requirement: claude plugin binary
The `orcai-plugins` repo SHALL contain a `plugins/claude/` directory with a `main.go` that wraps the `claude` CLI in non-interactive mode. The binary SHALL read the prompt from stdin, forward `ORCAI_MODEL` (or `--model` flag) to `claude --print`, and stream all stdout/stderr to its own stdout.

#### Scenario: Prompt delivered via stdin
- **WHEN** the binary is invoked as `orcai-claude` with a prompt on stdin
- **THEN** it spawns `claude --print` with stdin wired from the caller, stdout streamed back, and exits with claude's exit code

#### Scenario: Model flag forwarded
- **WHEN** `ORCAI_MODEL` env var is set to `claude-sonnet-4-6`
- **THEN** the binary passes `--model claude-sonnet-4-6` to the claude CLI

### Requirement: claude sidecar YAML
The `plugins/claude/` directory SHALL contain a `claude.yaml` sidecar file that users install to `~/.config/orcai/wrappers/claude.yaml`. It SHALL declare `name: claude`, `command: orcai-claude`, and a `models` list with at least the three current model IDs (opus-4-6, sonnet-4-6, haiku-4-5).

#### Scenario: Sidecar declares models
- **WHEN** the sidecar is loaded by `buildProviders`
- **THEN** the resulting `ProviderDef` has `Models` populated with the declared entries

#### Scenario: Sidecar installed to wrappers dir
- **WHEN** the file is present at `~/.config/orcai/wrappers/claude.yaml`
- **THEN** `discovery.Discover` returns a `TypeCLIWrapper` entry named `claude`
