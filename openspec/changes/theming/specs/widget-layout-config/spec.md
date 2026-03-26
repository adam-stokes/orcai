## MODIFIED Requirements

### Requirement: Layout config declares panes with widget and position
The layout config SHALL support a top-level `panes` list. Each entry SHALL specify at minimum: `name` (string, unique), `widget` (string, matches a known widget name), `position` (one of: `left`, `right`, `top`, `bottom`), and `size` (string, percentage or absolute columns/rows, e.g. `40%` or `80`). When computing panel body height, the switchboard SHALL subtract the rendered header height (which is variable when ANS sprites are active) rather than assuming a fixed 1-line title.

#### Scenario: Single pane declared and created
- **WHEN** `layout.yaml` declares one pane with `widget: welcome`, `position: left`, `size: 40%`
- **THEN** orcai creates a left split 40% wide in the current tmux window and launches the welcome widget in it

#### Scenario: Multiple panes created in declaration order
- **WHEN** `layout.yaml` declares two panes — sysop on the right at 60% and welcome on the left at 40%
- **THEN** orcai creates both panes and launches the respective widgets in each

#### Scenario: Invalid position value rejected
- **WHEN** a pane declares `position: diagonal`
- **THEN** orcai logs an error for that pane, skips it, and continues processing remaining panes

#### Scenario: Panel body height accounts for ANS sprite header
- **WHEN** the active theme bundle provides a 3-line ANS header sprite for a panel
- **THEN** the panel's body content area is 3 rows shorter than when using a 1-line plain-text title, and no content is clipped behind the header
