## ADDED Requirements

### Requirement: Box-drawing characters are the primary layout primitive
All screen section headers, content panels, and decorative dividers SHALL use Unicode box-drawing characters (`╔ ═ ╗ ║ ╠ ╣ ╚ ╝ ├ ┤ ─ │ ┼`) rendered in `<pre>` or inline `<span>` elements. CSS borders SHALL only supplement box-drawing art, not replace it.

#### Scenario: Section header rendered with box-drawing border
- **WHEN** any content section is rendered
- **THEN** the section title is enclosed in a box-drawing border using `╔═╗ ║ ╚═╝` or equivalent

#### Scenario: Dividers between content blocks use box-drawing
- **WHEN** two content blocks appear vertically adjacent
- **THEN** a `─` or `═` horizontal rule separates them, styled with a Dracula palette color

### Requirement: ANSI color classes map to Dracula palette variables
Color applied to ANSI art SHALL use CSS classes (`fg-purple`, `fg-cyan`, `fg-green`, `fg-yellow`, `fg-pink`, `fg-comment`, `fg-orange`) that resolve to CSS custom properties matching the Dracula palette. Hard-coded hex colors in ANSI art elements are NOT permitted.

#### Scenario: Box border uses palette class
- **WHEN** a box-drawing border is rendered
- **THEN** its color is set via a CSS class, not an inline `color:` style with a hex value

#### Scenario: Palette variables match Dracula spec
- **WHEN** `--purple` is used
- **THEN** it resolves to `#bd93f9` (Dracula purple)

### Requirement: Block art banners appear on the home and pipeline screens
The home screen SHALL display the existing ORCAI ANSI logo (`AnsiLogo` component). The pipeline screen SHALL display an ANSI DAG diagram showing a 3-step pipeline (Provider → Plugin → Output) using box-drawing and arrow characters (`→`, `▶`, `╌`).

#### Scenario: Home screen shows ANSI logo
- **WHEN** the home screen is active
- **THEN** the ORCAI ANSI logo is visible at the top of the content pane

#### Scenario: Pipeline screen shows ANSI DAG
- **WHEN** the Pipelines screen is active
- **THEN** an ANSI art diagram of a pipeline DAG is visible, using box-drawing characters for nodes and arrow characters for edges

### Requirement: Status bar is an ANSI-styled single-line bar
A persistent status bar at the bottom of the viewport SHALL be rendered as a full-width line using background fill (`▓` or CSS background) in a Dracula color. It SHALL display: node ID (`NODE-001`), active screen name, clock, and connection status (`● ONLINE`). It SHALL never scroll off screen.

#### Scenario: Status bar always visible
- **WHEN** the user scrolls the active pane
- **THEN** the status bar remains fixed at the bottom of the viewport

#### Scenario: Status bar shows active screen
- **WHEN** the user switches screens
- **THEN** the status bar updates to show the new screen name within 100ms

### Requirement: Pane headers use ANSI-style title bars
Each `.term-pane` SHALL have a title bar rendered as a box-drawing top border with the pane title embedded: `╔══[ PLUGINS ]══╗`. The title bar SHALL be sticky within the pane so it remains visible while the pane content scrolls.

#### Scenario: Pane title bar is sticky
- **WHEN** the user scrolls a content pane downward
- **THEN** the pane title bar (`╔══[ ... ]══╗`) remains at the top of the pane viewport
