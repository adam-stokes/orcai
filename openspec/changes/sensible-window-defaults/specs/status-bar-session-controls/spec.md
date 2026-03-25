## MODIFIED Requirements

### Requirement: Status bar shows session control hints
The tmux status bar right section SHALL display chord-key hints for new-session, prompt-builder, and window-navigation actions alongside the existing clock. The hints SHALL use the format `^spc n new  ^spc c win  ^spc p build` to communicate the `ctrl+space` chord prefix.

#### Scenario: Status bar contains new-session hint
- **WHEN** an orcai session is running
- **THEN** the tmux status bar right side contains a visible hint referencing the new-session chord

#### Scenario: Status bar contains window hint
- **WHEN** an orcai session is running
- **THEN** the tmux status bar right side contains a visible hint referencing the `c` chord for creating a window

#### Scenario: Status bar contains prompt-builder hint
- **WHEN** an orcai session is running
- **THEN** the tmux status bar right side contains a visible hint referencing the prompt-builder chord

#### Scenario: Clock remains visible
- **WHEN** an orcai session is running
- **THEN** the tmux status bar still shows the current time

## ADDED Requirements

### Requirement: Status bar displays a window indicator
The tmux status bar SHALL display the list of open windows using Dracula-themed formatting. Inactive windows SHALL use a muted color; the active window SHALL use bold foreground white. The indicator SHALL be placed in the status-left area, appended after the ORCAI label.

#### Scenario: Single window shown in status bar
- **WHEN** an orcai session has one window
- **THEN** the status bar shows `0:<window-name>` in the active-window style

#### Scenario: Multiple windows shown in status bar
- **WHEN** an orcai session has three windows
- **THEN** all three window index:name pairs appear in the status bar, with the current window highlighted in bold white and the others in muted purple

#### Scenario: New window appears in status bar immediately
- **WHEN** the user creates a new window via `ctrl+space c`
- **THEN** the new window's index and name appear in the status bar without requiring a manual refresh
