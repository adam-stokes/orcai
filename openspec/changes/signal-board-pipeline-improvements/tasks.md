## 1. Signal Board Directory Removal

- [x] 1.1 In `internal/switchboard/signal_board.go` `buildSignalBoard`, delete the `cwdSuffix` variable block (lines ~225–233) and remove `cwdSuffix` from the `rowContent` format string

## 2. Step Status Badge Table (Extended ASCII + Width Wrapping)

- [x] 2.1 In `internal/switchboard/switchboard.go`, define a `stepGlyph(status string) string` helper that maps `pending→·`, `running→»`, `done→°`, `failed→×`
- [x] 2.2 Replace the badge-rendering block (lines ~2250–2267) with a width-aware renderer: build each badge as `<color><glyph> <id><reset>`, measure visible width via `stripANSI`, append `  ·  ` separator between badges on the same row, start a new row when adding the next badge would exceed `width - 4`
- [x] 2.3 Add unit tests in `switchboard_test.go` (or `ansi_render_test.go`) covering: single-row case, multi-row wrap, all four glyph states

## 3. Pipeline Session Persistence

- [x] 3.1 In `internal/switchboard/jobwindow.go` `createJobWindow`, update `windowCmd` to append `; exec $SHELL` after the done-file write so the pane transitions to a live shell on completion

## 4. Step Failure Propagates to Feed Entry Status

- [x] 4.1 In `internal/switchboard/switchboard.go` `jobDoneMsg` handler (after drain loop, before `setFeedStatus`), add a loop over `m.feed[i].steps` to detect any step with `status == "failed"`; call `setFeedStatus(msg.id, FeedFailed)` instead of `FeedDone` when found
- [x] 4.2 Add a test in `switchboard_test.go` verifying that a `jobDoneMsg` on an entry with a failed step produces `FeedFailed`, and one with all-done steps produces `FeedDone`
