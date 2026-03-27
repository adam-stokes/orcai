## Context

Three UI subsystems need targeted fixes:

1. **Signal board** (`internal/switchboard/signal_board.go`) — `buildSignalBoard` appends a `cwdSuffix` to every row (lines 225–240). This clutters a space-constrained panel.

2. **Activity feed step badges** (`internal/switchboard/switchboard.go`, lines 2249–2267) — all step status badges are joined into one `badgeLine` with no width awareness. On pipelines with many steps the line overflows the terminal.

3. **Pipeline tmux window** (`internal/switchboard/jobwindow.go`, line 70) — `windowCmd` ends after the pipeline exits; `remain-on-exit` fires a "pane is dead" notice but gives no live shell. The user loses the ability to inspect the environment interactively.

4. **Feed entry status on step failure** (`switchboard.go`, line 737) — `jobDoneMsg` unconditionally sets `FeedDone` even when individual steps recorded `failed`. The signal board then shows a green ● for a partially-failed pipeline.

## Goals / Non-Goals

**Goals:**
- Remove directory suffix from signal board rows
- Wrap step badge row(s) to terminal width so no content goes off-screen
- Replace plain `[status] id` text badges with a compact table using CP437 extended ASCII characters (°, →, «, », ·, ─, etc.)
- Spawn an interactive `$SHELL` after pipeline completion so the tmux window stays alive and usable
- Promote feed entry to `FeedFailed` when any step has `status: "failed"` at `jobDoneMsg` time

**Non-Goals:**
- Removing the cwd display from the activity feed (only the signal board row)
- Changing pipeline YAML format or step execution order
- Persisting shell across agent runner windows (pipeline-only change)

## Decisions

### 1. Signal board: drop cwdSuffix entirely

Remove the `cwdSuffix` variable and its interpolation from `buildSignalBoard`. The cwd remains visible in the activity feed detail view; the signal board row doesn't need it.

*Alternative considered*: truncate cwd to N chars. Rejected — the title + status + timestamp already fill the row; any cwd only steals space.

### 2. Step badge table: wrap to width, use extended ASCII glyphs

Replace the single joined `badgeLine` with a width-aware renderer that:
- Builds each badge as `«status-glyph» id` where status glyphs are:
  - pending: `·` (U+00B7 / CP437 250)
  - running: `»` (U+00BB / CP437 175)
  - done: `°` (U+00B0 / CP437 248)
  - failed: `×` (U+00D7 / CP437 158)
- Measures visible width (stripping ANSI) before appending each badge to the current row
- Starts a new continuation line (indented 2 spaces) when adding the next badge would exceed `width - 4`
- Separates badges on the same line with `  ·  ` (middle-dot spacer)

*Alternative considered*: a full box-drawing bordered table (╔═╗). Rejected — too tall for dense pipelines with many steps; compact inline rows are more scannable.

### 3. Pipeline session persistence: exec $SHELL after done file

Change `windowCmd` to append `; exec $SHELL` after writing the done file:

```
{ <cmd> ; } 2>&1 | tee <log> ; echo $? > <done> ; exec $SHELL
```

`exec $SHELL` replaces the current shell process, so the pane transitions seamlessly to an interactive shell in the same working directory. `remain-on-exit` stays set as a safety net in case `$SHELL` itself exits.

*Alternative considered*: write a wrapper script. Rejected — the inline approach is simpler and requires no temp file.

### 4. Step failure → FeedFailed promotion

In the `jobDoneMsg` handler, after draining buffered lines, scan `m.feed[i].steps` for any step with `status == "failed"`. If found, call `setFeedStatus(msg.id, FeedFailed)` instead of `FeedDone`.

*Alternative considered*: promote at `StepStatusMsg` time. Rejected — a step can transiently fail and be retried; final status should only be locked in at job completion.

## Risks / Trade-offs

- **exec $SHELL changes CWD behavior**: `exec $SHELL` inherits the subshell's cwd (which is `startDir` for the window). This is the correct behavior — the user lands in the pipeline's working directory. No mitigation needed.
- **Badge wrap depends on `stripANSI`**: if new ANSI sequences are added to badge colors, the width measurement must keep using the stripped version. Current `stripANSI` regex is a reliable dependency.
- **Step failure promotion is final**: once `FeedFailed` is set, a subsequent retry-succeeded message won't auto-clear it. This is acceptable for v1; a future "retry" feature would need to reset step state explicitly.

## Migration Plan

All changes are local to `internal/switchboard/`. No config changes, no wire format changes, no migration needed. A `make build` and restart is sufficient.
