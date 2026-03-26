## Why

The ORCAI switchboard has a `themes` package with a full Registry, Bundle, and YAML schema — but none of it is wired into the UI. Every panel header, status bar, border, and popup renders hardcoded Dracula escape codes in `switchboard.go` and `styles/styles.go`, making the look completely frozen. Users who want a different aesthetic (or the BBS-faithful ANSI art feel) have no path forward.

## What Changes

- **Wire `themes.Registry` into the switchboard** so palette colors, border styles, and status bar formats actually apply at render time instead of hardcoded ANSI constants.
- **Expand `themes.Bundle`** to support per-section ANSI art header files (`headers.pipelines`, `headers.agent_runner`, `headers.signal_board`, `headers.activity_feed`) and a configurable popup/modal ANS sprite.
- **Build an ANSI header renderer** that reads `.ans` files from the active theme bundle and renders them in place of the current plain-text section titles.
- **Ship four original `.ans` header sprites** in the ABS bundled theme — one per switchboard panel — crafted in authentic IBM PC ANSI/VT100 style with block-character backgrounds and color glows.
- **Add a Theme Switcher overlay popup** triggered by a keybind in the switchboard, presenting a scrollable list of available themes with live preview of palette and any bundle splash art.
- **Make popup/modal styling themeable** via `Bundle.Modal` (background, border color, title bar color), applied to the quit-confirm, agent launch, and theme-switcher overlays.
- **Store user themes in `~/.config/orcai/themes/<name>/theme.yaml`** alongside existing config, fully hackable.

## Capabilities

### New Capabilities
- `ansi-section-headers`: Per-panel ANSI art header rendering — `.ans` files in a theme bundle replace the plain-text `PIPELINES`, `AGENT RUNNER`, `SIGNAL BOARD`, `ACTIVITY FEED` titles. Includes the ANS parser/renderer and the four original ABS sprites.
- `theme-switcher-popup`: Live theme switching overlay — scrollable picker listing all Registry bundles, shows palette swatches and splash preview, applies the chosen theme instantly via `bus.Publish(TopicThemeChanged)`.
- `switchboard-theme-wiring`: Connect `themes.Registry` to the switchboard model so all lipgloss styles, border colors, status bar format/colors, and ANSI palette constants are driven by the active `Bundle` instead of hardcoded values.

### Modified Capabilities
- `widget-layout-config`: Panel header height must accommodate ANS sprite height (variable, ≥1 line); layout calculations need to read rendered header height instead of assuming a fixed 1-line title.

## Impact

- `internal/themes/themes.go` — add `Headers` (map of panel→ANS path) and `Modal` struct to `Bundle`
- `internal/switchboard/switchboard.go` — consume `*themes.Bundle` from a Registry stored on the model; derive lipgloss styles and ANSI palette from active bundle; subscribe to `TopicThemeChanged` bus events
- `internal/styles/styles.go` — replace hardcoded color vars with functions that accept a `*themes.Bundle`
- `internal/assets/themes/abs/` — add four `.ans` header sprites and a modal sprite
- New `internal/switchboard/ansi_render.go` — `.ans` file parser and terminal renderer
- New `internal/switchboard/theme_picker.go` — theme switcher overlay model and view
- No new external dependencies; existing `lipgloss`, `bubbletea`, `busd` bus are sufficient
