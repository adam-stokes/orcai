## MODIFIED Requirements

### Requirement: Dashboard is persistent and does not auto-exit
Window 0 SHALL display the live dashboard indefinitely after launch. The dashboard SHALL NOT exit on arbitrary keypresses. Only `q`, `esc`, or `ctrl+c` SHALL close the dashboard. Pressing `n` or `enter` SHALL open the provider/model picker popup and then close the dashboard. Pressing `p` SHALL open the prompt builder in a new tmux window without closing the dashboard. All other advertised chord keys SHALL execute their corresponding tmux operation without closing the dashboard.

#### Scenario: Dashboard stays open after launch
- **WHEN** orcai starts and window 0 opens
- **THEN** the dashboard remains visible and does not exit until the user presses `q`, `esc`, or `ctrl+c`

#### Scenario: n opens picker and closes dashboard
- **WHEN** the user presses `n`
- **THEN** the provider/model picker popup opens and the dashboard closes

#### Scenario: enter opens picker and closes dashboard
- **WHEN** the user presses `enter`
- **THEN** the provider/model picker popup opens and the dashboard closes

#### Scenario: q closes the dashboard
- **WHEN** the user presses `q`
- **THEN** the dashboard exits cleanly

#### Scenario: esc closes the dashboard
- **WHEN** the user presses `esc`
- **THEN** the dashboard exits cleanly

#### Scenario: Session-level keys do not close dashboard
- **WHEN** the user presses `t`, `d`, `c`, `|`, `-`, or any arrow key
- **THEN** the dashboard remains open

## ADDED Requirements

### Requirement: Session-level chord keys execute their tmux operations from the welcome widget
The dashboard SHALL execute the following operations when the corresponding key is pressed, without closing the dashboard:
- `t`: run `orcai sysop` to open the sysop panel
- `d`: detach the tmux client (`tmux detach-client`)
- `c`: create a new plain tmux window (`tmux new-window`)
- `|`: split the current pane right (`tmux split-window -h`)
- `-`: split the current pane down (`tmux split-window -v`)
- Arrow keys (`Up`, `Down`, `Left`, `Right`): navigate to the adjacent pane

#### Scenario: d detaches the tmux client
- **WHEN** the user presses `d`
- **THEN** `tmux detach-client` is called and the session remains running

#### Scenario: c creates a new window
- **WHEN** the user presses `c`
- **THEN** a new tmux window is created and focus shifts to it; the dashboard remains open in its pane

#### Scenario: | splits pane right
- **WHEN** the user presses `|`
- **THEN** `tmux split-window -h` is called and a new right-side pane appears

#### Scenario: - splits pane down
- **WHEN** the user presses `-`
- **THEN** `tmux split-window -v` is called and a new bottom pane appears

#### Scenario: Arrow keys navigate panes
- **WHEN** the user presses an arrow key
- **THEN** focus moves to the adjacent pane in that direction

#### Scenario: p opens prompt builder in new window
- **WHEN** the user presses `p`
- **THEN** a new tmux window named `prompt-builder` opens running the prompt builder; the dashboard remains open
