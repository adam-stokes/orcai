## Why

The welcome widget shows a full chord reference but none of those keys do anything — only `q`, `esc`, and `enter`/`n` are handled. A user sitting on the welcome screen should be able to take any of the advertised actions directly without reaching for `ctrl+space` first. The welcome panel is the home screen; it should behave like one.

## What Changes

- `n` / `enter` → open provider picker popup (start new session)
- `t` → toggle sysop panel (run `orcai sysop` via tmux run-shell)
- `p` → open prompt builder in a new window
- `d` → detach tmux client (session stays alive)
- `c` → create a new plain shell window
- `|` → split current pane right
- `-` → split current pane down
- Arrow keys → navigate to adjacent pane
- `q` / `esc` / `ctrl+c` → close the welcome widget (already works, unchanged)

Actions that don't require closing the welcome widget (`t`, `d`, `c`, `|`, `-`, arrows) are executed as fire-and-forget tmux commands while the widget stays open. Actions that open a new UI (`n`, `enter`, `p`) close the widget first.

The model gains a `self string` field (path to the orcai binary) so it can construct tmux commands for `t` and `p` without hardcoding binary names.

## Capabilities

### New Capabilities

*(none)*

### Modified Capabilities

- `welcome-dashboard`: Key handling in `Update` extended to cover all advertised chord actions; model gains `self string` for constructing orcai subcommand invocations

## Impact

- `internal/welcome/welcome.go`: `model` struct, `Update`, `newModel`, new `tmuxExec` helper
- No changes to UI, bus logic, or tests outside of welcome
