## 1. Extend actionMap with window/pane actions

- [x] 1.1 Add `new-window`, `prev-window`, `next-window` entries to `actionMap` in `internal/keybindings/keybindings.go`
- [x] 1.2 Add `split-pane-right`, `split-pane-down`, `kill-pane` entries to `actionMap`
- [x] 1.3 Add `select-pane-left`, `select-pane-right`, `select-pane-up`, `select-pane-down` entries to `actionMap`

## 2. Add chord bindings to buildTmuxConf

- [x] 2.1 Bind `c` in `orcai-chord` to `new-window` in `buildTmuxConf` (`internal/bootstrap/bootstrap.go`)
- [x] 2.2 Bind `[` and `]` in `orcai-chord` to `previous-window` and `next-window`
- [x] 2.3 Bind `|` and `-` in `orcai-chord` to `split-window -h` and `split-window -v`
- [x] 2.4 Bind arrow keys (`Left`, `Right`, `Up`, `Down`) in `orcai-chord` to the corresponding `select-pane` directions
- [x] 2.5 Bind `x` in `orcai-chord` to `kill-pane`

## 3. Restore window indicator in status bar

- [x] 3.1 Set `window-status-format` in `buildTmuxConf` to `#[fg=#6272a4] #I:#W ` (muted, inactive)
- [x] 3.2 Set `window-status-current-format` to `#[fg=#f8f8f2,bold] #I:#W ` (white bold, active)
- [x] 3.3 Expand `status-left-length` to accommodate the window list alongside the ORCAI label
- [x] 3.4 Append `#{W:#{window-status-format},#{window-status-current-format}}` (or equivalent tmux format) to `status-left`

## 4. Update status-right hint text

- [x] 4.1 Update `status-right` string in `buildTmuxConf` to include `^spc c win` hint alongside existing hints

## 5. Tests

- [x] 5.1 Add unit tests for new `actionMap` entries in `internal/keybindings/keybindings_test.go` (verify each new action resolves to expected tmux args)
- [x] 5.2 Add/update `buildTmuxConf` tests in `internal/bootstrap/bootstrap_test.go` to assert new chord bindings are present in the generated config string
- [x] 5.3 Assert `window-status-format` and `window-status-current-format` are non-empty in the generated config
