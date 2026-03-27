## Why

The signal board and pipeline output pane have several UX rough edges: the directory path clutters the signal board, step status badges overflow the viewport, the pipeline session exits after completion (losing shell access), and the step status display lacks visual polish. These issues compound on every pipeline run and impede day-to-day usability.

## What Changes

- Remove the current working directory display from the signal board entry line
- Wrap or truncate the step status row in the activity feed so it does not overflow the terminal width
- Replace the plain `[done]`/`[running]`/`[failed]` step status badges with a compact, visually-distinct table using box-drawing and extended ASCII characters (degrees °, arrows →, angle quotes « », middle dots ·, dashes —, etc.) from the CP437/ANSI 128–255 range
- Keep the pipeline tmux session alive after a run completes (success or failure) so the shell remains accessible

## Capabilities

### New Capabilities

- `signal-board-display`: Controls what fields are rendered per entry in the signal board widget — remove directory, retain timestamp + name + status
- `pipeline-step-status-table`: A formatted inline table of step statuses using extended ASCII characters, rendered inside the activity feed
- `pipeline-session-persistence`: Pipeline tmux sessions remain open after run completion, presenting a live shell prompt

### Modified Capabilities

- `pipeline-step-lifecycle`: Step status rendering changes — badges must wrap/truncate to terminal width and adopt the new table format

## Impact

- `internal/ui/signalboard/` — remove directory field from rendered row
- `internal/ui/feed/` or wherever step status badges are rendered — wrap/truncate logic + new ASCII table renderer
- `internal/runner/` or pipeline execution shell script — remove `exit` / add persistent shell after pipeline completes
- No API or protocol changes; purely UI and session lifecycle
