## ADDED Requirements

### Requirement: Signal board rows omit the working directory
The signal board panel SHALL NOT display the working directory path in any feed entry row. Only the cursor indicator, LED status dot, timestamp, title, and status label SHALL appear on each row.

#### Scenario: Entry with cwd renders without directory
- **WHEN** a feed entry has a non-empty `cwd` field
- **THEN** the signal board row shows only `[led] HH:MM:SS  <title>  <status>` with no path suffix

#### Scenario: Entry without cwd renders normally
- **WHEN** a feed entry has an empty `cwd` field
- **THEN** the signal board row renders identically to one with cwd (no change in output)
