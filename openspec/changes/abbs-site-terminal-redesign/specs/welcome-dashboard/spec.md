## MODIFIED Requirements

### Requirement: Dashboard displays ANSI/BBS banner header
The existing `buildWelcomeArt` banner SHALL be displayed at the top of the dashboard. The banner SHALL use the Dracula palette (purple `#6272a4` / `#bd93f9`, pink `#ff79c6`, teal) and box-drawing characters (`╔ ═ ╗ ║ ╠ ╣ ╚ ╝`). The banner SHALL scale to terminal width. The banner text SHALL read "ORCAI — ABBS" (Agentic Bulletin Board System) reflecting the current product branding; the old "ABS" short form SHALL NOT appear.

#### Scenario: Banner renders at top
- **WHEN** the dashboard view is rendered
- **THEN** the first lines contain the ORCAI banner with box-drawing borders and the "ORCAI — ABBS" logotype

#### Scenario: Banner scales to terminal width
- **WHEN** the terminal is resized
- **THEN** the banner redraws at the new width without truncation or overflow

#### Scenario: Banner uses ABBS branding
- **WHEN** the dashboard banner text is rendered
- **THEN** the subtitle line reads "ABBS" or "Agentic Bulletin Board System", not "ABS" or "Agentic Bulletin System"

### Requirement: Footer shows chord-key hints
The dashboard footer SHALL display `^spc n new · ^spc p build` hints in dim blue, consistent with the status-bar hints. The footer SHALL also display `q quit · enter connect` as additional hints, reflecting that the dashboard is the primary entry point to the ABBS workspace.

#### Scenario: Footer shows navigation hints
- **WHEN** the dashboard is rendered
- **THEN** the footer contains `^spc n new`, `^spc p build`, `q quit`, and `enter connect`
