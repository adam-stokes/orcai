## 1. Fix key handling in Update

- [x] 1.1 Replace the `case tea.KeyMsg:` block in `internal/welcome/welcome.go` with an explicit switch: `q`/`esc`/`ctrl+c` → quit; `enter` → launchPicker + quit; all other keys → no-op (do not quit)

## 2. Update help text in buildHelp

- [x] 2.1 Replace the footer line `"  ── enter new session · any key continue ──"` with `"  enter  new session   (pick provider + model)"` and `"  q / esc  close"`
- [x] 2.2 Add new chord entries after the existing `d  detach` line: `c  new shell window`, `|  split right   -  split down`, `←→↑↓  navigate panes`

## 3. Tests

- [x] 3.1 In the welcome package tests, verify that pressing a non-special key (e.g. `a`) does NOT cause the model to quit (check that `Update` returns a nil cmd and `launchPicker` is false)
- [x] 3.2 Verify that pressing `q` causes `Update` to return `tea.Quit`
- [x] 3.3 Verify that pressing `esc` causes `Update` to return `tea.Quit`
- [x] 3.4 Verify that pressing `enter` sets `launchPicker = true` and returns `tea.Quit`
- [x] 3.5 Verify that `buildHelp` output contains `q / esc` and does NOT contain `any key continue`
- [x] 3.6 Verify that `buildHelp` output contains `c  new shell window`
