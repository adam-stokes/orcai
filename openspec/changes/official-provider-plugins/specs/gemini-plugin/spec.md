## ADDED Requirements

### Requirement: gemini plugin binary
The `orcai-plugins` repo SHALL contain a `plugins/gemini/` directory with a `main.go` that wraps the `gemini` CLI. The binary SHALL read the prompt from stdin, pass `ORCAI_MODEL` as the model flag if set, and stream the response to stdout.

#### Scenario: Prompt delivered via stdin
- **WHEN** the binary is invoked as `orcai-gemini` with a prompt on stdin
- **THEN** it spawns the gemini CLI in non-interactive mode, streams stdout back, and exits with gemini's exit code

#### Scenario: Model flag forwarded
- **WHEN** `ORCAI_MODEL` is set to `gemini-2.0-flash`
- **THEN** the binary passes the appropriate model flag to the gemini CLI

### Requirement: gemini sidecar YAML
The `plugins/gemini/` directory SHALL contain a `gemini.yaml` sidecar that users install to `~/.config/orcai/wrappers/gemini.yaml`. It SHALL declare `name: gemini`, `command: orcai-gemini`, and a `models` list containing at minimum `gemini-2.0-flash` and `gemini-1.5-pro`.

#### Scenario: Sidecar declares models
- **WHEN** the sidecar is loaded by `buildProviders`
- **THEN** the resulting `ProviderDef` has `Models` with at least `gemini-2.0-flash` and `gemini-1.5-pro`
