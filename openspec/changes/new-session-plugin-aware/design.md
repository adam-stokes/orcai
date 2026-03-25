## Context

The New Session picker (`internal/picker/`) currently populates its `— providers —` section from a bundled `providers.Registry` that hard-codes six provider profiles and checks `exec.LookPath` for each binary. This worked before the plugin system existed but now diverges from runtime reality: plugins are registered in a `plugin.Manager` whose category index already tracks availability. The `orcai new` command (`cmd/new.go`) is a TODO stub, so selecting any item from the picker produces no session at all. Pipelines are listed but have no launch path either.

## Goals / Non-Goals

**Goals:**
- Picker provider section driven by plugin Manager: show a provider only when its plugin is registered OR its binary exists in PATH
- Pipelines launch a real shell session: new tmux window running `orcai pipeline run <file>` so the user sees live output
- `orcai new` implemented: reads a serialised `PickerItem` (passed via stdin or `ORCAI_PICKER_SELECTION` env) and creates the appropriate tmux window
- No ghost entries: if nothing installs Claude, the Claude row disappears

**Non-Goals:**
- Changing pipeline execution internals or the plugin Manager architecture
- Adding new provider integrations
- Changing how skills or agents appear in the picker (those sections are already dynamic)
- Redesigning the picker UI layout or search behaviour

## Decisions

### Decision 1: Provider discovery via plugin Manager category index + PATH fallback

**Choice:** At picker startup, call `manager.ListByCategory("providers")` to get all registered provider plugins. For each bundled provider profile whose plugin is _not_ in the Manager, fall back to `exec.LookPath(profile.Binary)` — if found, create a transient `CliAdapter` and include it.

**Rationale:** The Manager is already the runtime source of truth. Using it avoids duplicating detection logic. The PATH fallback preserves behaviour for setups that haven't yet migrated to explicit plugin registration (e.g. user has `claude` in PATH but no sidecar YAML). Keeping both paths means zero regression for current users.

**Alternative considered:** Scan `~/.config/orcai/plugins/` and `~/.config/orcai/wrappers/` directly at picker startup. Rejected — bypasses Manager dedup/override logic and re-implements discovery that `discovery.go` already handles.

### Decision 2: Pipeline sessions launch `orcai pipeline run` in a new tmux window

**Choice:** When a pipeline `PickerItem` is selected, `orcai new` opens a new tmux window with the command `orcai pipeline run <pipeline-file>`. The window name is set to the pipeline name. The shell session stays open after the pipeline completes so the user can review output.

**Rationale:** Pipelines are long-running, streaming operations. Embedding execution inside the picker or the `new` command would block the picker popup and lose the tmux window model. Launching a real shell session means the user gets scrollback, can re-run, and the dashboard can pick up telemetry events from that window.

**Alternative considered:** Run the pipeline inline in the popup and then open a results pane. Rejected — too complex, breaks the uniform "each window = one session" model, and makes error recovery harder.

### Decision 3: `orcai new` receives selection as JSON via env var

**Choice:** The picker writes the selected `PickerItem` as JSON to `ORCAI_PICKER_SELECTION` env var before exiting (or to stdout for `tmux display-popup` capture). `orcai new` reads and unmarshals this to decide what to launch.

**Rationale:** `display-popup` captures the picker's stdout already; passing the full item struct as JSON is simpler than a positional arg scheme that would need escaping for file paths, model IDs, etc. The existing picker already returns text to the caller — we replace that with structured JSON.

**Alternative considered:** Pass individual flags (`--kind`, `--provider`, `--pipeline-file`). Rejected — fragile as `PickerItem` fields grow; JSON round-trips cleanly.

## Risks / Trade-offs

- **Plugin Manager not initialised at picker startup** → Mitigation: picker startup calls `discovery.Discover(configDir)` + `manager.LoadWrappersFromDir` before building items, same flow as pipeline run
- **Ollama model injection** → Current code injects discovered Ollama models into the provider list; this logic must be preserved in the new discovery path. Mitigation: keep a dedicated Ollama discovery pass in `BuildProviders`
- **`orcai new` replaces current picker return value** → Any existing callers that parse the picker's stdout plain-text output will break. Mitigation: `welcome.go` is the only caller; update it at the same time

## Open Questions

- Should `orcai pipeline run` use `--no-exit` or similar to keep the window open after completion, or rely on the shell staying alive? Need to confirm tmux `remain-on-exit` setting in the orcai session config.
