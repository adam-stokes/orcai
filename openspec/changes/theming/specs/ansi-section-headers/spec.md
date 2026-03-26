## ADDED Requirements

### Requirement: Bundle may declare per-panel ANS header files
The `themes.Bundle` struct SHALL support an optional `Headers` field containing a map of panel names to relative `.ans` file paths within the bundle directory. Valid panel keys are `pipelines`, `agent_runner`, `signal_board`, and `activity_feed`. If a panel key is absent, the plain-text title SHALL be rendered for that panel.

#### Scenario: Bundle with headers field loads ANS bytes
- **WHEN** `theme.yaml` declares `headers.pipelines: headers/pipelines.ans` and the file exists in the bundle directory
- **THEN** `Bundle.HeaderBytes["pipelines"]` contains the raw bytes of that file after loading

#### Scenario: Missing header file is silently skipped
- **WHEN** `theme.yaml` declares a header path but the file does not exist in the bundle
- **THEN** the bundle loads successfully and `Bundle.HeaderBytes["pipelines"]` is nil; the panel renders its plain-text title

#### Scenario: Bundle without headers field is valid
- **WHEN** `theme.yaml` has no `headers:` section
- **THEN** the bundle loads without error and all panels render plain-text titles

### Requirement: ANS renderer emits a reset after sprite bytes
The ANS renderer SHALL append `\x1b[0m` (SGR reset) after emitting the raw sprite bytes to prevent escape state from leaking into subsequent UI elements.

#### Scenario: Reset is appended to sprite output
- **WHEN** the renderer renders a panel header from `[]byte` ANS data
- **THEN** the output ends with `\x1b[0m`

### Requirement: ANS header falls back to plain text when terminal is too narrow
The renderer SHALL compare the sprite's column width (derived by scanning for the longest line in the `.ans` data) against the current terminal width. If the terminal is narrower than the sprite, the plain-text panel title SHALL be rendered instead.

#### Scenario: Terminal narrower than sprite falls back
- **WHEN** the switchboard terminal width is 60 and the sprite's widest line is 80 columns
- **THEN** the panel renders its plain-text title ("PIPELINES", etc.) instead of the ANS sprite

#### Scenario: Terminal wide enough renders sprite
- **WHEN** the switchboard terminal width is 120 and the sprite's widest line is 80 columns
- **THEN** the ANS sprite bytes are emitted for that panel header

### Requirement: ABS bundled theme ships four header sprites
The `internal/assets/themes/abs/` bundle SHALL include header sprites for all four switchboard panels: `headers/pipelines.ans`, `headers/agent_runner.ans`, `headers/signal_board.ans`, and `headers/activity_feed.ans`. Each sprite SHALL be 3 lines tall and use CP437 block characters with standard 8/16-color ANSI sequences.

#### Scenario: All four ABS header files exist after build
- **WHEN** the binary is compiled
- **THEN** `assets.ThemeFS` contains all four ANS files under `themes/abs/headers/`

#### Scenario: ABS header sprites are non-empty
- **WHEN** `LoadBundled()` is called
- **THEN** the ABS bundle's `HeaderBytes` map contains non-empty byte slices for all four panel keys
