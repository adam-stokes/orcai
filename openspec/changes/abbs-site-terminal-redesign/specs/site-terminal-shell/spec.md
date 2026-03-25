## ADDED Requirements

### Requirement: Site renders as a single-page terminal emulator shell
The site SHALL be delivered as a single HTML document. All navigation between screens SHALL occur by showing/hiding `<section data-screen="...">` elements in-place. No browser navigation events, page reloads, or URL changes SHALL occur when switching screens.

#### Scenario: Switching screens does not reload the page
- **WHEN** the user presses a screen shortcut key (e.g., `2` for Getting Started)
- **THEN** the new screen content becomes visible and the previous screen is hidden without any network request or page reload

#### Scenario: All screen content present in initial HTML
- **WHEN** the browser loads the site for the first time
- **THEN** the HTML document contains all screen sections; only the active screen is visible

### Requirement: Keyboard router dispatches global and per-screen shortcuts
A `KeyboardRouter` singleton SHALL be initialized on page load. It SHALL handle:
- Keys `1`–`7` to switch to screens by index
- `?` or `F1` to open the keyboard help overlay
- `q` or `Escape` to trigger the "disconnect" fade animation
- Delegation of additional keys to a per-screen handler registered for the active screen

#### Scenario: Numeric key switches screen
- **WHEN** the user presses `3` while viewing any screen
- **THEN** the third screen (Plugins) becomes active and the nav indicator updates

#### Scenario: Help overlay opens on `?`
- **WHEN** the user presses `?`
- **THEN** a full-screen keyboard help overlay appears listing all global and screen-local shortcuts

#### Scenario: Help overlay closes on `?` or `Escape`
- **WHEN** the help overlay is open and the user presses `?` or `Escape`
- **THEN** the overlay closes and the previous screen is restored

#### Scenario: Keyboard input ignored when text field focused
- **WHEN** focus is inside an `<input>` or `<textarea>` element
- **THEN** the keyboard router does not intercept keypresses

### Requirement: Navigation indicator reflects active screen
The nav bar SHALL display the active screen label highlighted (CSS class `active`). The node status bar SHALL display the current screen name. Both SHALL update synchronously when the screen changes.

#### Scenario: Active nav item highlighted
- **WHEN** the user switches to the Pipelines screen
- **THEN** the "Pipelines" nav item has the `active` class and all others do not

#### Scenario: Status bar shows screen name
- **WHEN** the Plugins screen is active
- **THEN** the status bar shows `SCREEN: PLUGINS`

### Requirement: Body viewport does not scroll; content panes scroll internally
`<body>` SHALL have `overflow: hidden`. Each screen's `.term-pane` content area SHALL have `overflow-y: auto` and a fixed height calculated to fill the viewport minus the header and status bar. Scrollbar styling SHALL match the terminal aesthetic (thin, dark, Dracula-palette thumb).

#### Scenario: Body scroll locked
- **WHEN** the user scrolls the mouse wheel while viewing any screen
- **THEN** the body does not scroll; only the active `.term-pane` scrolls

#### Scenario: Pane scrolls independently
- **WHEN** the active screen's content exceeds the pane height
- **THEN** the pane scrollbar appears and the user can scroll within the pane

### Requirement: Noscript fallback displays all screens stacked
When JavaScript is disabled, all `<section data-screen="...">` elements SHALL be visible and stacked vertically. A `<noscript>` banner SHALL inform the user that keyboard navigation requires JavaScript.

#### Scenario: Content visible without JS
- **WHEN** a browser with JS disabled loads the site
- **THEN** all screen sections are visible and readable without interaction
