## ADDED Requirements

### Requirement: Theme switcher overlay opens via keybind
The switchboard SHALL open a theme switcher overlay when the user presses `T` (capital T) while no other overlay is open. The overlay SHALL render as a centered floating panel on top of the switchboard using the existing `overlayCenter` helper.

#### Scenario: T key opens theme picker
- **WHEN** the switchboard has focus and the user presses `T`
- **THEN** the theme switcher overlay becomes visible and key events are routed to it

#### Scenario: T key does not open picker when another overlay is active
- **WHEN** the agent modal or quit-confirm overlay is already open
- **THEN** pressing `T` has no effect

### Requirement: Theme switcher lists all available bundles
The overlay SHALL display a scrollable list of all `themes.Registry` bundles showing each bundle's `DisplayName`, a row of palette color swatches (seven colored blocks representing BG, FG, Accent, Dim, Border, Error, Success), and the bundle name in parentheses.

#### Scenario: All bundles appear in the list
- **WHEN** the registry contains three bundles
- **THEN** the overlay lists all three with their display names

#### Scenario: Active theme is highlighted
- **WHEN** the overlay is open
- **THEN** the currently active bundle row is visually highlighted (inverted colors or selection background)

### Requirement: Selecting a theme applies it live
Pressing `enter` on a highlighted bundle SHALL call `registry.SetActive(name)`, persist the choice, publish a `TopicThemeChanged` bus event, and close the overlay. The switchboard SHALL repaint with the new theme without requiring a restart.

#### Scenario: Enter applies the selected theme
- **WHEN** the user navigates to a bundle row and presses `enter`
- **THEN** the registry's active theme is updated, the overlay closes, and the switchboard re-renders with the new theme's colors

#### Scenario: Theme choice is persisted across restarts
- **WHEN** the user selects a non-default theme and restarts orcai
- **THEN** the same theme is active on the next launch

### Requirement: Theme switcher is dismissible without changing theme
Pressing `esc` or `q` in the theme switcher SHALL close the overlay without changing the active theme.

#### Scenario: Esc closes picker without applying
- **WHEN** the theme switcher is open and the user presses `esc`
- **THEN** the overlay closes and the active theme is unchanged

### Requirement: Bundle modal config styles all overlay popups
The `themes.Bundle` SHALL support an optional `Modal` block with fields `bg`, `border`, `title_bg`, and `title_fg`. These SHALL be applied to the quit-confirm modal, agent-launch modal, and theme-switcher overlay. Fields may be palette references (e.g. `palette.accent`). Absent fields SHALL fall back to the existing hardcoded Dracula defaults.

#### Scenario: Modal bg color comes from bundle
- **WHEN** a bundle sets `modal.bg: "#1e1e2e"`
- **THEN** all overlay popups render their background with that color

#### Scenario: Missing modal block uses defaults
- **WHEN** a bundle has no `modal:` section
- **THEN** overlays render using the existing default colors (no regression)
