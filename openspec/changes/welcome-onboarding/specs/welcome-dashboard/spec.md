## MODIFIED Requirements

### Requirement: Dashboard is persistent and does not auto-exit
Window 0 SHALL display the live dashboard indefinitely after launch. The dashboard SHALL NOT exit on arbitrary keypresses. Only `q`, `esc`, or `ctrl+c` SHALL close the dashboard. Pressing `enter` SHALL open the provider/model picker popup and then close the dashboard.

#### Scenario: Dashboard stays open after launch
- **WHEN** orcai starts and window 0 opens
- **THEN** the dashboard remains visible and does not exit until the user presses `q`, `esc`, or `ctrl+c`

#### Scenario: Arbitrary keypresses do not close dashboard
- **WHEN** the user presses any key other than `q`, `esc`, `ctrl+c`, or `enter`
- **THEN** the dashboard remains open and does not exit

#### Scenario: q closes the dashboard
- **WHEN** the user presses `q`
- **THEN** the dashboard exits cleanly

#### Scenario: esc closes the dashboard
- **WHEN** the user presses `esc`
- **THEN** the dashboard exits cleanly

### Requirement: Footer shows chord-key hints
The dashboard footer SHALL display the full current chord reference including window and pane management bindings added in `sensible-window-defaults`. The footer SHALL show a `q / esc  close` hint instead of "any key continue".

#### Scenario: Footer shows navigation hints
- **WHEN** the dashboard is rendered
- **THEN** the footer contains `^spc n new` and `^spc p build` entries

#### Scenario: Footer shows window/pane hints
- **WHEN** the dashboard is rendered
- **THEN** the footer contains entries for `c` (new shell window), `|` (split right), `-` (split down), and arrow key pane navigation

#### Scenario: Footer shows close hint not any-key hint
- **WHEN** the dashboard is rendered
- **THEN** the footer contains `q / esc  close` and does NOT contain the text "any key continue"

#### Scenario: Enter hint is explicit in the help body
- **WHEN** the dashboard is rendered
- **THEN** the help text contains an explicit `enter  new session` entry
