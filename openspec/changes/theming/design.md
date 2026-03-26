## Context

The `internal/themes` package already defines `Bundle`, `Palette`, `Borders`, `StatusBar`, a `Registry` with persistence, and a `TopicThemeChanged` bus event — but nothing in the UI actually consumes it. The switchboard renders using hardcoded ANSI escape constants (`aPur`, `aCyn`, etc.) and `internal/styles/styles.go` has hardcoded Dracula `lipgloss.Color` values. The ABS bundled theme has a `splash.ans` field but it is never read or rendered.

This design wires everything together: connect the Registry to the switchboard, render `.ans` header sprites per-panel, add a live theme switcher overlay, and expand the `Bundle` schema to cover all theming surfaces (headers, modals).

## Goals / Non-Goals

**Goals:**
- Every color, border style, and status bar value rendered by the switchboard is driven by the active `themes.Bundle`.
- Per-panel ANSI art headers (`.ans` files) replace the plain text `PIPELINES` / `AGENT RUNNER` / `SIGNAL BOARD` / `ACTIVITY FEED` titles when a bundle provides them.
- A theme switcher overlay (keybind `T` from the switchboard) lists all Registry bundles; selecting one applies the theme live via bus event, no restart needed.
- The ABS bundled theme ships four authentic ANSI/CP437 header sprites and a modal sprite.
- User themes live at `~/.config/orcai/themes/<name>/theme.yaml` alongside other config, fully hackable.
- Modal/popup styling (background, border, title bar) is controlled by a `Bundle.Modal` config block.

**Non-Goals:**
- Re-theming widgets outside the switchboard (chatui, gitui, etc.) in this change.
- An interactive ANS art editor.
- Font or terminal font configuration.
- Any networked or cloud theme repository.

## Decisions

### 1. Registry lives on the switchboard model

The switchboard `Model` struct gets a `*themes.Registry` field. On `Init`, it calls `themes.NewRegistry(userThemesDir)`. The `Active()` bundle is read at render time via a helper that converts `Bundle` fields into `lipgloss` styles and raw ANSI strings.

**Alternatives considered:**
- Global singleton registry — rejected because it makes testing harder and is hidden state.
- Passing bundle through tea.Cmd — over-engineered; the registry is small and reads are cheap.

### 2. Styles are derived at render time, not pre-built globals

`internal/styles/styles.go` currently exports package-level `lipgloss.Style` vars initialized at import time. We replace these with functions accepting `*themes.Bundle` (e.g. `styles.TitleStyle(b)`) that construct the style on the fly from `b.Palette`. The hardcoded ANSI palette constants in `switchboard.go` are replaced with helpers that map `Bundle.Palette` fields to ANSI 24-bit escape sequences.

**Alternatives considered:**
- Keeping the global vars and mutating them on theme change — racy and hard to test.
- A `StyleSheet` struct built once per theme — adds complexity with no clear benefit given BubbleTea's per-frame render model.

### 3. ANS header rendering: raw byte passthrough with line-count introspection

`.ans` files are read at load time from the bundle's embedded or user FS into `[]byte`. At render time the bytes are emitted verbatim inside the panel title zone; terminal supports the embedded escape sequences natively. Before emitting we count `\n` bytes to know the sprite height so layout calculations can accommodate variable-height headers.

**Bundle schema extension:**
```yaml
headers:
  pipelines: headers/pipelines.ans
  agent_runner: headers/agent_runner.ans
  signal_board: headers/signal_board.ans
  activity_feed: headers/activity_feed.ans
modal:
  bg: palette.bg
  border: palette.accent
  title_bg: palette.accent
  title_fg: palette.bg
```

**Alternatives considered:**
- Parsing ANS into a structured IR — overkill; the terminal already knows how to render escape sequences.
- Using Sixel or iTerm2 inline images — not universally supported; ANSI block characters work everywhere.

### 4. Theme switcher: BubbleTea overlay model (same pattern as agent modal)

A new `themePicker` sub-model lives in `internal/switchboard/theme_picker.go`. It mirrors the existing `agentModal` pattern: a boolean `themePickerOpen` on the switchboard model gates rendering and key handling. The picker renders as a centered floating box (via the existing `overlayCenter` helper) showing a numbered list of bundles with a palette swatch row and the bundle's `DisplayName`. Pressing `enter` calls `registry.SetActive(name)` and publishes `TopicThemeChanged` to the bus.

**Alternatives considered:**
- A dedicated BubbleTea program spawned in a tmux pane — too heavy; the switchboard already has overlay infrastructure.
- A separate CLI command — not live; would require restart.

### 5. Bus event drives live re-render

The switchboard subscribes to `TopicThemeChanged` during `Init`. On receipt it reads `registry.Active()` and returns `tea.Batch(tea.ClearScreen, tickCmd())` to force a full repaint with the new theme. No model restart required.

### 6. ABS bundled ANS sprites: CP437 block art, 8-color ANSI

The four header sprites use IBM CP437 block characters (█ ▄ ▀ ░ ▒ ▓), box-drawing lines, and standard 8/16-color ANSI sequences (no 24-bit) for maximum terminal compatibility. Each sprite is exactly 3 lines tall so layout overhead is predictable. The modal sprite is 1 line (a decorative title bar). Sprites are authored as raw `.ans` text files committed to `internal/assets/themes/abs/headers/`.

## Risks / Trade-offs

- **Terminal width mismatch** → ANS sprites are authored at 80 columns. On narrow terminals they may wrap. Mitigation: the renderer will measure terminal width and skip the sprite (fall back to plain text title) if `termWidth < spriteWidth`.
- **ANSI escape bleed** → a malformed `.ans` file could leave escape state open and corrupt subsequent UI. Mitigation: the renderer always emits `\x1b[0m` (reset) after the sprite bytes.
- **`styles.go` API is a breaking change** → callers outside the switchboard that import `styles.TitleStyle` (a var) will break. Mitigation: audit callers at change time; there are currently very few.
- **Registry load on switchboard init** → first launch is slightly slower if the user has many theme files. In practice theme directories will be small (< 10); no caching needed.

## Migration Plan

1. Expand `Bundle` struct in `themes.go` (additive; existing YAML files continue to parse, new fields default to zero).
2. Update `styles.go` to export style-factory functions alongside (then in place of) the old vars.
3. Wire Registry into the switchboard model; thread `bundle` through render helpers.
4. Add ANS renderer + theme picker overlay.
5. Add ABS sprites to `internal/assets/themes/abs/headers/`.
6. Update loader to read `headers.*` ANS files from the bundle FS and store bytes on `Bundle`.

No config migration required. The existing `active_theme` persistence file stays compatible.

## Open Questions

- Should the theme switcher show a live palette preview row of color swatches, or is a text list sufficient for v1? (Current answer: swatch row using `lipgloss` colored blocks — simple and visually useful.)
- Should `modal.bg` apply to ALL overlays (quit-confirm, agent-launch, theme-picker), or only the theme-picker? (Current answer: all overlays — consistency is better UX.)
