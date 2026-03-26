## ADDED Requirements

### Requirement: Switchboard model holds a themes Registry
The switchboard `Model` struct SHALL contain a `*themes.Registry` field initialized during `New()` using user themes from `~/.config/orcai/themes/`. All color and style rendering SHALL derive values from `registry.Active()` at render time rather than hardcoded constants.

#### Scenario: Switchboard initializes with registry
- **WHEN** `switchboard.New()` is called
- **THEN** the model's registry field is non-nil and `registry.Active()` returns a valid bundle

#### Scenario: Panel border color comes from active bundle
- **WHEN** the active bundle sets `palette.border: "#ff79c6"`
- **THEN** all switchboard panel borders render in that color

### Requirement: Styles package exports bundle-aware factory functions
`internal/styles/styles.go` SHALL export functions (e.g. `TitleStyle(b *themes.Bundle) lipgloss.Style`) that accept a `*themes.Bundle` and return a `lipgloss.Style` derived from the bundle's palette. The existing package-level vars SHALL be removed or deprecated.

#### Scenario: TitleStyle uses bundle accent color
- **WHEN** `styles.TitleStyle(b)` is called with a bundle whose `palette.accent` is `"#ff0000"`
- **THEN** the returned style has foreground color `#ff0000`

### Requirement: Switchboard subscribes to TopicThemeChanged
The switchboard SHALL subscribe to `themes.TopicThemeChanged` on the bus during `Init`. On receiving the event it SHALL call `registry.Active()` to obtain the new bundle and return a `tea.Cmd` that triggers a full repaint.

#### Scenario: Theme change event triggers repaint
- **WHEN** a `theme.changed` event is published on the bus
- **THEN** the switchboard re-renders within one BubbleTea update cycle using the new theme's colors

### Requirement: Status bar format and colors come from the active bundle
The status bar SHALL render using `Bundle.StatusBar.Format` as the template string, `Bundle.StatusBar.BG` as background color, and `Bundle.StatusBar.FG` as foreground color. Palette references (e.g. `palette.accent`) SHALL be resolved via `Bundle.ResolveRef`.

#### Scenario: Status bar uses bundle format
- **WHEN** the active bundle sets `statusbar.format: " {session} · {model} "`
- **THEN** the status bar renders that format string with the session and model tokens substituted

#### Scenario: Status bar uses bundle colors
- **WHEN** the active bundle sets `statusbar.bg: palette.bg` and `statusbar.fg: palette.accent`
- **THEN** the status bar background is the bundle's BG color and foreground is the Accent color

### Requirement: User theme directory is scanned at startup
Orcai SHALL scan `~/.config/orcai/themes/` for user-provided theme bundles on every launch, merging them into the Registry (user wins on name collision with bundled themes).

#### Scenario: User theme overrides bundled theme of same name
- **WHEN** `~/.config/orcai/themes/abs/theme.yaml` exists with different palette values
- **THEN** the registry uses the user-provided ABS bundle instead of the embedded one

#### Scenario: Empty user theme directory loads cleanly
- **WHEN** `~/.config/orcai/themes/` does not exist or is empty
- **THEN** the switchboard starts with only the bundled themes and no error is emitted
