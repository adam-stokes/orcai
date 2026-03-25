## 1. Add self-path to model and tmuxExec helper

- [x] 1.1 Add `self string` field to `model` struct in `internal/welcome/welcome.go`
- [x] 1.2 Add `resolveSelf() string` function that calls `os.Executable()` + `filepath.EvalSymlinks`
- [x] 1.3 Set `self: resolveSelf()` in `newModel()`
- [x] 1.4 Add `tmuxExec(args ...string) tea.Cmd` helper that runs `tmux <args>` as a fire-and-forget `tea.Cmd` returning nil

## 2. Extend key handling in Update

- [x] 2.1 Add case `"n"` as an alias for `"enter"` (set `m.launchPicker = true; return m, tea.Quit`)
- [x] 2.2 Add case `"t"` → `return m, tmuxExec("run-shell", m.self+" sysop")`
- [x] 2.3 Add case `"p"` → `return m, tmuxExec("new-window", "-n", "prompt-builder", m.self+" _promptbuilder")`
- [x] 2.4 Add case `"d"` → `return m, tmuxExec("detach-client")`
- [x] 2.5 Add case `"c"` → `return m, tmuxExec("new-window")`
- [x] 2.6 Add case `"|"` → `return m, tmuxExec("split-window", "-h")`
- [x] 2.7 Add case `"-"` → `return m, tmuxExec("split-window", "-v")`
- [x] 2.8 Add arrow key cases using `msg.Type`: `tea.KeyUp/Down/Left/Right` → `tmuxExec("select-pane", "-U/-D/-L/-R")`

## 3. Tests

- [x] 3.1 Verify pressing `n` sets `launchPicker = true` and returns `tea.Quit` (same behavior as `enter`)
- [x] 3.2 Verify pressing `t`, `p`, `d`, `c`, `|`, `-` each return a non-nil cmd but do NOT set `launchPicker` and do NOT return `tea.Quit`
- [x] 3.3 Verify pressing an arrow key returns a non-nil cmd and does not quit
- [x] 3.4 Verify pressing an unhandled key (e.g. `z`) returns nil cmd and does not quit
