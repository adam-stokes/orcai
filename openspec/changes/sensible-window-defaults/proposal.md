## Why

Orcai launches inside tmux but completely suppresses window/pane management: the status bar hides window names, no chord bindings exist for splitting or navigating panes, and users who want a plain shell window or a side-by-side view have no keyboard path to get there. Users should be able to do what tmux naturally provides — create windows, split panes, switch between them — without memorising raw tmux commands.

## What Changes

- Add built-in chord bindings for window operations: new window, next/previous window, kill window
- Add built-in chord bindings for pane operations: split right, split down, navigate panes (arrow keys), kill pane
- Restore window name display in the tmux status bar (Dracula-themed, minimal)
- Add new `actionMap` entries in `keybindings.go` so user `keybindings.yaml` can override or extend these operations
- Update `status-right` hint text to reflect the new window/pane chord keys

## Capabilities

### New Capabilities

- `window-pane-defaults`: Built-in chord keybindings and status bar display for tmux window and pane management (create, split, navigate, kill); no user config required

### Modified Capabilities

- `status-bar-session-controls`: Status bar right section gains a window indicator; window-status-format strings are restored to show window index and name in Dracula palette

## Impact

- `internal/bootstrap/bootstrap.go`: `buildTmuxConf` adds window-status format strings and new chord bindings
- `internal/keybindings/keybindings.go`: `actionMap` gains window/pane action entries
- No breaking changes; all new bindings use previously-unbound chord keys
