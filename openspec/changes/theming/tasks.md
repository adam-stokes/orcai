## 1. Expand themes.Bundle schema

- [x] 1.1 Add `Headers map[string]string` field to `Bundle` (keys: pipelines, agent_runner, signal_board, activity_feed; values: relative .ans paths)
- [x] 1.2 Add `Modal` struct to `Bundle` with fields `BG`, `Border`, `TitleBG`, `TitleFG string` (palette refs supported)
- [x] 1.3 Add `HeaderBytes map[string][]byte` to `Bundle` (populated by loader, not YAML)
- [x] 1.4 Update `loader.go` `loadFromFS` and `LoadUser` to read header `.ans` files from the bundle FS/dir into `HeaderBytes` after YAML unmarshal
- [x] 1.5 Update `themes_test.go` to assert ABS bundle HeaderBytes are non-nil for all four panel keys

## 2. Author ABS bundled ANS header sprites

- [x] 2.1 Research 16colo.rs ANSI art conventions and CP437 block-character palette; document the approach in a brief comment at top of each .ans file
- [x] 2.2 Create `internal/assets/themes/abs/headers/pipelines.ans` — 3-line CP437 block art title, purple/cyan glow, "PIPELINES" wordmark with block-character background
- [x] 2.3 Create `internal/assets/themes/abs/headers/agent_runner.ans` — 3-line sprite, green/purple tones, "AGENT RUNNER" wordmark
- [x] 2.4 Create `internal/assets/themes/abs/headers/signal_board.ans` — 3-line sprite, cyan accent, "SIGNAL BOARD" wordmark with block-char radio-wave motif
- [x] 2.5 Create `internal/assets/themes/abs/headers/activity_feed.ans` — 3-line sprite, pink/yellow tones, "ACTIVITY FEED" wordmark with scrolling-lines motif
- [x] 2.6 Create `internal/assets/themes/abs/headers/modal.ans` — 1-line decorative title bar sprite for popups
- [x] 2.7 Update `internal/assets/themes/abs/theme.yaml` to declare all six header paths under `headers:` and `modal.splash: headers/modal.ans`

## 3. ANS header renderer

- [x] 3.1 Create `internal/switchboard/ansi_render.go` with `RenderHeader(bundle *themes.Bundle, panel string, termWidth int) string` function
- [x] 3.2 Implement sprite column-width detection (scan .ans bytes for longest line, counting printable rune widths, ignoring escape sequences)
- [x] 3.3 Implement fallback to plain-text panel title when termWidth < sprite width or HeaderBytes[panel] is nil
- [x] 3.4 Ensure `\x1b[0m` is appended after any sprite bytes in the output
- [x] 3.5 Write unit tests for `RenderHeader`: nil bytes falls back, narrow terminal falls back, wide terminal returns sprite + reset

## 4. Styles package refactor

- [x] 4.1 Add bundle-aware factory functions to `internal/styles/styles.go`: `TitleStyle`, `SubtitleStyle`, `SelectedStyle`, `DimmedStyle`, `NormalStyle`, `SuccessStyle`, `ErrorStyle`, `WarningStyle`, `BorderStyle` — each accepts `*themes.Bundle`
- [x] 4.2 Add ANSI palette helper `BundleANSI(b *themes.Bundle) ANSIPalette` that returns a struct of pre-formatted `\x1b[38;2;R;G;Bm` sequences derived from `b.Palette`
- [x] 4.3 Keep existing global vars temporarily (mark `// Deprecated`) so other callers don't break; plan their removal in a follow-up

## 5. Wire themes.Registry into the switchboard

- [x] 5.1 Add `registry *themes.Registry` field to switchboard `Model`
- [x] 5.2 In `New()`, call `themes.NewRegistry(userThemesDir)` where `userThemesDir` is `~/.config/orcai/themes`; store on model
- [x] 5.3 Replace hardcoded `aPur`, `aCyn`, `aPnk`, `aGrn`, `aRed`, `aSelBg` constants in panel render functions with calls to `BundleANSI(m.registry.Active())`
- [x] 5.4 Replace hardcoded lipgloss styles in `viewPipelines`, `viewAgentRunner`, `viewSignalBoard`, `viewActivityFeed`, `viewStatusBar` with bundle-aware factory calls
- [x] 5.5 Replace panel plain-text titles with calls to `RenderHeader(bundle, panelKey, m.termWidth)` in each panel view function
- [x] 5.6 Subscribe to `themes.TopicThemeChanged` in `Init` via the bus; on receipt update `registry.active` and return `tea.ClearScreen` cmd
- [x] 5.7 Apply `Bundle.Modal` colors to `viewQuitModalBox` and `viewAgentModalBox` border and title bar styles

## 6. Theme switcher overlay

- [x] 6.1 Create `internal/switchboard/theme_picker.go` with `themePicker` struct: fields `bundles []themes.Bundle`, `cursor int`
- [x] 6.2 Implement `viewThemePicker(m Model, w int) string` that renders a centered box listing bundles with DisplayName, name-in-parens, and a swatch row of seven colored `█` blocks (BG, FG, Accent, Dim, Border, Error, Success)
- [x] 6.3 Add `themePickerOpen bool` to switchboard `Model`
- [x] 6.4 Handle `T` keypress in switchboard key handler: open picker when no other overlay is active
- [x] 6.5 Route `j`/`k`/`up`/`down` and `enter`/`esc`/`q` to picker when `themePickerOpen`; enter calls `registry.SetActive` + publishes bus event + closes picker; esc/q closes without change
- [x] 6.6 Add `overlayCenter(base, m.viewThemePicker(w), w, h)` render path in `View()` when `themePickerOpen`
- [x] 6.7 Write tests for theme picker: opens on T, closes on esc, entry count matches registry, enter triggers theme change

## 7. Switchboard test updates

- [x] 7.1 Update existing switchboard tests that assert hardcoded color strings to use bundle-derived expected values
- [x] 7.2 Add test: switchboard renders ABS header sprite bytes in panel title zone when termWidth is sufficient
- [x] 7.3 Add test: switchboard falls back to plain-text title when no HeaderBytes for panel

## 8. Verification

- [x] 8.1 Run `go test ./internal/themes/... ./internal/switchboard/... ./internal/styles/...` — all pass
- [ ] 8.2 Launch orcai, press `T`, cycle through themes, confirm colors update live without restart
- [ ] 8.3 Verify ANS header sprites render in the four panels with no escape bleed on a standard 80-column and 120-column terminal
- [ ] 8.4 Add a user theme at `~/.config/orcai/themes/mytest/theme.yaml`, restart orcai, confirm it appears in the picker
