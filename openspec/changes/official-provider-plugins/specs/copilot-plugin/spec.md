## ADDED Requirements

### Requirement: github-copilot plugin binary
The `orcai-plugins` repo SHALL contain a `plugins/github-copilot/` directory with a `main.go` that wraps `gh copilot suggest` in non-interactive mode. The binary SHALL read the prompt from stdin and write the copilot response to stdout.

#### Scenario: Prompt delivered via stdin
- **WHEN** the binary is invoked as `orcai-github-copilot` with a prompt on stdin
- **THEN** it spawns `gh copilot suggest -t shell` (or equivalent non-interactive flag), streams stdout back, and exits with gh's exit code

### Requirement: github-copilot sidecar YAML
The `plugins/github-copilot/` directory SHALL contain a `github-copilot.yaml` sidecar that users install to `~/.config/orcai/wrappers/github-copilot.yaml`. It SHALL declare `name: github-copilot` and `command: orcai-github-copilot`. A `models` list MAY be empty if Copilot does not expose model selection via its CLI.

#### Scenario: Sidecar discovered
- **WHEN** the file is present at `~/.config/orcai/wrappers/github-copilot.yaml`
- **THEN** `discovery.Discover` returns a `TypeCLIWrapper` entry named `github-copilot` and the agent runner shows it as an available provider
