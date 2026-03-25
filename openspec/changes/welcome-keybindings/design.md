## Context

`internal/welcome/welcome.go` has a BubbleTea `Update` that currently handles three cases in `tea.KeyMsg`: quit keys, `enter`/`n` (open picker), and everything else (no-op). The `model` has `pickerCmd []string` for the picker launch and a `launchPicker bool` flag. After the BubbleTea program exits, `Run()` checks `launchPicker` and fires a `tmux display-popup`.

The advertised chord actions in `buildHelp` cover two categories:
1. **Session-level tmux ops** (`c`, `|`, `-`, arrows, `t`, `d`) — run a tmux command; welcome stays open
2. **UI-launching ops** (`n`/`enter`, `p`) — close welcome first, then open the new UI

## Goals / Non-Goals

**Goals:**
- All keys shown in `buildHelp` do what they say
- Session-level ops run without closing the welcome widget
- UI-launching ops close welcome then act
- Keep code changes minimal and within `welcome.go`

**Non-Goals:**
- Visual feedback/animation on keypress
- Key repeat or hold-key behavior
- Changing what any key does outside the welcome widget

## Decisions

### 1. Fire-and-forget tmux helper

For session-level ops, add a helper that returns a `tea.Cmd` executing `tmux <args>` asynchronously:

```go
func tmuxExec(args ...string) tea.Cmd {
    return func() tea.Msg {
        exec.Command("tmux", args...).Run() //nolint:errcheck
        return nil
    }
}
```

Returning `nil` as the message means BubbleTea ignores the result. The welcome widget stays open and responsive.

**Alternative considered:** Run tmux synchronously in `Update`. Rejected — BubbleTea's `Update` should be non-blocking.

### 2. Self-path in model

Add `self string` to `model`, resolved in `newModel()` via `os.Executable()` + `filepath.EvalSymlinks`. This lets `t` and `p` construct `orcai sysop` / `orcai _promptbuilder` invocations correctly regardless of installation path.

```go
func resolveSelf() string {
    self, _ := os.Executable()
    if resolved, err := filepath.EvalSymlinks(self); err == nil {
        return resolved
    }
    return self
}
```

### 3. Post-quit action for prompt builder

`p` needs to open a new tmux window AND exit the welcome widget (so the user's focus shifts to the new window). Use the same post-quit pattern as `launchPicker`: store a `postQuitCmd []string` on the model, executed in `Run()` after `p.Run()` returns.

Actually, `p` can fire the tmux command without quitting — `tmux new-window` creates the window and shifts focus. The welcome pane stays open. This is consistent with the `c` key behavior. Use `tmuxExec` for `p` as well.

**Revised**: `p` uses `tmuxExec("new-window", "-n", "prompt-builder", m.self+" _promptbuilder")`. The `_promptbuilder` subcommand name mirrors the existing chord binding in `bootstrap.go`.

### 4. Key assignments in Update

```
n / enter  → m.launchPicker = true; tea.Quit
t          → tmuxExec("run-shell", m.self+" sysop")
p          → tmuxExec("new-window", "-n", "prompt-builder", m.self+" _promptbuilder")
d          → tmuxExec("detach-client")
c          → tmuxExec("new-window")
|          → tmuxExec("split-window", "-h")
-          → tmuxExec("split-window", "-v")
Up         → tmuxExec("select-pane", "-U")
Down       → tmuxExec("select-pane", "-D")
Left       → tmuxExec("select-pane", "-L")
Right      → tmuxExec("select-pane", "-R")
q/esc/ctrl+c → tea.Quit (unchanged)
```

## Risks / Trade-offs

- **`_promptbuilder` subcommand**: The chord binding uses `_promptbuilder`; if that internal command doesn't exist, `p` silently fails. Acceptable — same behavior as the chord today.
- **`sysop` without toggle**: `orcai sysop` opens the panel; it doesn't toggle. If it's already open, a second invocation opens another instance. This is a pre-existing limitation of the sysop command, not introduced here.
- **Arrow keys in BubbleTea**: `tea.KeyUp`, `tea.KeyDown`, etc. need to be matched by `KeyType`, not by string. The `msg.Type` field is used for special keys.

## Migration Plan

No config, binary, or API changes. Takes effect on next `make run`.

## Open Questions

*(none)*
