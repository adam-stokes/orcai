## 1. Plugin-Aware Provider Discovery

- [x] 1.1 Add `BuildProviders(mgr *plugin.Manager, configDir string)` function in `internal/picker/` that queries Manager for `providers.*` category plugins and falls back to PATH check for bundled profiles
- [x] 1.2 Preserve Ollama model injection in the new discovery path (enumerate discovered Ollama models and inject as provider items)
- [x] 1.3 Omit the `— providers —` section header entirely when no providers are discovered
- [x] 1.4 Replace the existing `BuildProviders()` call in `picker.go` with the new plugin-aware version

## 2. PickerItem JSON Serialisation

- [x] 2.1 Change picker exit path to serialise the selected `PickerItem` as JSON into `ORCAI_PICKER_SELECTION` env var (via `tmux setenv` or stdout capture) instead of plain-text output
- [x] 2.2 Ensure `PipelineFile` field is set on pipeline `PickerItem`s when building the pipeline section
- [x] 2.3 Update `welcome.go` caller to read `ORCAI_PICKER_SELECTION` JSON and pass it to `orcai new` instead of parsing plain-text picker stdout

## 3. Implement orcai new Command

- [x] 3.1 Implement `cmd/new.go`: read and unmarshal `ORCAI_PICKER_SELECTION` env var; return error if `$TMUX` is unset
- [x] 3.2 Handle `kind == "provider"`: resolve provider profile from plugin Manager or registry, build tmux window-name from `SessionConfig.WindowName` template, launch binary with `SessionConfig.LaunchArgs` in new tmux window
- [x] 3.3 Handle `kind == "pipeline"`: open new tmux window running `orcai pipeline run <pipelineFile>`; set window name to pipeline name; ensure shell remains open after run completes
- [x] 3.4 Handle `kind == "session"` / empty fallback: open new tmux window running `$SHELL`
- [x] 3.5 Return informative error and non-zero exit code for unknown provider IDs

## 4. Tests

- [x] 4.1 Unit tests for new `BuildProviders`: manager-registered provider appears, PATH-only provider appears via fallback, uninstalled provider omitted, empty section when nothing installed
- [x] 4.2 Unit tests for `orcai new` launch logic: provider dispatch, pipeline dispatch, shell fallback, not-in-tmux error
- [x] 4.3 Update existing picker tests that assert on provider items to use the new discovery path

## 5. Validation

- [ ] 5.1 Manual smoke test: install `claude` binary, open picker, confirm Claude appears; uninstall/rename binary, confirm Claude disappears
- [ ] 5.2 Manual smoke test: select a pipeline from the picker, confirm new tmux window opens running `orcai pipeline run`
- [x] 5.3 Run `go test ./internal/picker/... ./cmd/...` and confirm all pass
