## ADDED Requirements

### Requirement: orcai new launches a provider session
`orcai new` SHALL read a `PickerItem` JSON from the `ORCAI_PICKER_SELECTION` environment variable. When `kind == "provider"`, it SHALL open a new tmux window in the current orcai session, set the window name from the provider's `SessionConfig.WindowName` template, and run the provider binary with `SessionConfig.LaunchArgs`.

#### Scenario: Provider item launches tmux window
- **WHEN** `ORCAI_PICKER_SELECTION` contains a `PickerItem` with `kind: "provider"` and `providerID: "claude"`
- **THEN** `orcai new` creates a new tmux window named per the Claude profile template and runs the Claude binary

#### Scenario: Unknown provider returns error
- **WHEN** `ORCAI_PICKER_SELECTION` references a `providerID` not found in the plugin Manager or provider registry
- **THEN** `orcai new` exits with a non-zero status and prints an error message

### Requirement: orcai new launches a pipeline session
When `kind == "pipeline"`, `orcai new` SHALL open a new tmux window and run `orcai pipeline run <pipelineFile>` in that window. The window name SHALL be set to the pipeline's `Name` field. The window SHALL remain open (shell stays alive) after the pipeline process exits.

#### Scenario: Pipeline item launches orcai pipeline run
- **WHEN** `ORCAI_PICKER_SELECTION` contains a `PickerItem` with `kind: "pipeline"` and `pipelineFile: "~/.config/orcai/pipelines/my-pipeline.pipeline.yaml"`
- **THEN** `orcai new` opens a new tmux window running `orcai pipeline run ~/.config/orcai/pipelines/my-pipeline.pipeline.yaml`

#### Scenario: Pipeline window name set to pipeline name
- **WHEN** a pipeline item with `name: "code-review"` is launched
- **THEN** the new tmux window is named `code-review`

### Requirement: orcai new launches a plain shell session
When `kind == "session"` or when no `ORCAI_PICKER_SELECTION` is set, `orcai new` SHALL open a new tmux window running `$SHELL`.

#### Scenario: Shell fallback when no selection
- **WHEN** `ORCAI_PICKER_SELECTION` is unset or empty
- **THEN** `orcai new` opens a new tmux window running `$SHELL`

### Requirement: orcai new returns error when not inside tmux
`orcai new` SHALL return a non-zero exit code and print a descriptive error if it is not running inside a tmux session (i.e. `$TMUX` is unset).

#### Scenario: Not inside tmux
- **WHEN** `$TMUX` environment variable is unset
- **THEN** `orcai new` prints "must be run inside a tmux session" and exits with code 1
