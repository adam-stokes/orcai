## ADDED Requirements

### Requirement: Chord bindings for window creation and navigation are active by default
Orcai SHALL bind the following keys in the `orcai-chord` tmux key table without requiring any `keybindings.yaml` file:
- `c` → create a new tmux window
- `[` → switch to the previous window
- `]` → switch to the next window

#### Scenario: New window created with chord
- **WHEN** the user presses `ctrl+space` then `c` inside an orcai session
- **THEN** a new tmux window is created and focused

#### Scenario: Navigate to previous window
- **WHEN** the user presses `ctrl+space` then `[` and a previous window exists
- **THEN** tmux switches focus to the previous window

#### Scenario: Navigate to next window
- **WHEN** the user presses `ctrl+space` then `]` and a next window exists
- **THEN** tmux switches focus to the next window

#### Scenario: Bindings active without keybindings.yaml
- **WHEN** `~/.config/orcai/keybindings.yaml` does not exist
- **THEN** the window navigation chord bindings are still active in the session

### Requirement: Chord bindings for pane splitting are active by default
Orcai SHALL bind the following keys in the `orcai-chord` tmux key table without requiring any `keybindings.yaml` file:
- `|` → split the current pane vertically (side-by-side)
- `-` → split the current pane horizontally (top/bottom)

#### Scenario: Split pane right with chord
- **WHEN** the user presses `ctrl+space` then `|`
- **THEN** the current pane is split vertically and the new right-side pane is focused

#### Scenario: Split pane down with chord
- **WHEN** the user presses `ctrl+space` then `-`
- **THEN** the current pane is split horizontally and the new bottom pane is focused

### Requirement: Chord bindings for pane navigation are active by default
Orcai SHALL bind arrow keys in the `orcai-chord` tmux key table to move focus between panes in the corresponding direction.

#### Scenario: Navigate to left pane
- **WHEN** the user presses `ctrl+space` then the left arrow key
- **THEN** focus moves to the pane to the left of the current pane (if one exists)

#### Scenario: Navigate to right pane
- **WHEN** the user presses `ctrl+space` then the right arrow key
- **THEN** focus moves to the pane to the right of the current pane (if one exists)

#### Scenario: Navigate to pane above
- **WHEN** the user presses `ctrl+space` then the up arrow key
- **THEN** focus moves to the pane above the current pane (if one exists)

#### Scenario: Navigate to pane below
- **WHEN** the user presses `ctrl+space` then the down arrow key
- **THEN** focus moves to the pane below the current pane (if one exists)

### Requirement: Chord binding for killing the current pane is active by default
Orcai SHALL bind `x` in the `orcai-chord` tmux key table to kill the current pane.

#### Scenario: Kill current pane with chord
- **WHEN** the user presses `ctrl+space` then `x`
- **THEN** the current tmux pane is destroyed

#### Scenario: Single-pane window survives kill
- **WHEN** the user presses `ctrl+space` then `x` and the current pane is the only pane in the window
- **THEN** tmux's default behavior applies (window closes if last pane; session remains)

### Requirement: Window/pane operations are exposed as named actions in actionMap
The `keybindings.go` `actionMap` SHALL include named actions for all built-in window/pane operations so users can remap them via `keybindings.yaml`:
`new-window`, `prev-window`, `next-window`, `split-pane-right`, `split-pane-down`, `kill-pane`, `select-pane-left`, `select-pane-right`, `select-pane-up`, `select-pane-down`.

#### Scenario: User remaps new-window via keybindings.yaml
- **WHEN** `keybindings.yaml` contains `key: "M-c"` with `action: new-window`
- **THEN** orcai binds `Alt+c` to create a new window at startup
