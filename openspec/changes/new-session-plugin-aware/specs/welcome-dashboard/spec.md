## MODIFIED Requirements

### Requirement: Picker provider section is dynamically populated from installed plugins
The picker's `— providers —` section SHALL be populated at startup by querying the plugin Manager for all plugins registered under the `providers` category, plus a PATH-fallback check for bundled provider profiles whose binary is in PATH but not yet registered as a plugin. Providers that are not installed (no plugin registration and binary not found) SHALL NOT appear in the list.

#### Scenario: Installed provider plugin appears in picker
- **WHEN** a plugin is registered in the Manager under the `providers` category with name `claude`
- **THEN** the picker shows a "Claude" entry in the providers section

#### Scenario: Binary-in-PATH provider appears via fallback
- **WHEN** no `providers.claude` plugin is registered but `claude` binary is found via `exec.LookPath`
- **THEN** the picker shows a "Claude" entry in the providers section

#### Scenario: Uninstalled provider does not appear
- **WHEN** neither a `providers.opencode` plugin is registered nor the `opencode` binary is in PATH
- **THEN** the picker shows no OpenCode entry

#### Scenario: Empty providers section when nothing installed
- **WHEN** no provider plugins are registered and no provider binaries are in PATH
- **THEN** the `— providers —` section header is omitted from the picker

## ADDED Requirements

### Requirement: Pipeline picker items carry pipelineFile path
Each pipeline `PickerItem` built by the picker SHALL set the `PipelineFile` field to the absolute path of the `.pipeline.yaml` file, so that `orcai new` can pass it directly to `orcai pipeline run`.

#### Scenario: Pipeline item includes file path
- **WHEN** the picker discovers a pipeline at `~/.config/orcai/pipelines/foo.pipeline.yaml`
- **THEN** the resulting `PickerItem` has `kind: "pipeline"` and `pipelineFile` set to that absolute path

### Requirement: Picker selection is serialised as JSON to ORCAI_PICKER_SELECTION
When the user selects an item, the picker SHALL serialise the selected `PickerItem` as JSON and write it to `ORCAI_PICKER_SELECTION` before exiting, replacing the previous plain-text stdout output.

#### Scenario: Selection written as JSON env var
- **WHEN** the user selects a provider item
- **THEN** the picker exits and `ORCAI_PICKER_SELECTION` contains a valid JSON-encoded `PickerItem`
