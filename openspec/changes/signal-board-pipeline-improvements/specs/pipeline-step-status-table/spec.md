## ADDED Requirements

### Requirement: Step status badges use extended ASCII glyphs
Each step status badge in the activity feed SHALL use a CP437/extended ASCII glyph to represent the step state:
- `pending`: `·` (middle dot, U+00B7)
- `running`: `»` (right double angle quotation mark, U+00BB)
- `done`: `°` (degree sign, U+00B0)
- `failed`: `×` (multiplication sign, U+00D7)

The badge format SHALL be `<color><glyph> <step-id><reset>`.

#### Scenario: Done step renders degree glyph
- **WHEN** a step has status `done`
- **THEN** the badge renders as `° <step-id>` in the success color

#### Scenario: Running step renders double-angle glyph
- **WHEN** a step has status `running`
- **THEN** the badge renders as `» <step-id>` in the accent color

#### Scenario: Failed step renders multiplication glyph
- **WHEN** a step has status `failed`
- **THEN** the badge renders as `× <step-id>` in the error color

#### Scenario: Pending step renders middle-dot glyph
- **WHEN** a step has status `pending` or any unrecognized status
- **THEN** the badge renders as `· <step-id>` in the dim color

### Requirement: Step badge rows wrap to terminal width
The step badge display in the activity feed SHALL wrap across multiple rows rather than overflowing the terminal width. When appending the next badge to the current row would cause the visible (ANSI-stripped) line length to exceed `width - 4`, a new continuation row SHALL begin (indented 2 spaces). Badges on the same row SHALL be separated by `  ·  ` (two spaces, middle dot, two spaces) in the dim color.

#### Scenario: Few steps fit on one row
- **WHEN** a pipeline has 3 steps and their badges fit within `width - 4` visible characters
- **THEN** all badges appear on a single indented row

#### Scenario: Many steps wrap to multiple rows
- **WHEN** a pipeline has enough steps that their badges exceed `width - 4` visible characters
- **THEN** badges wrap onto additional continuation rows, each indented 2 spaces, with no badge truncated or omitted

#### Scenario: Single very long step id fits on its own row
- **WHEN** a single step id is longer than `width - 8` characters
- **THEN** it occupies its own row without being truncated
