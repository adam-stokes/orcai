## MODIFIED Requirements

### Requirement: Feed entry status reflects step-level failures at job completion
When a `jobDoneMsg` is received (pipeline process exited with code 0), the runner SHALL inspect all recorded steps for the feed entry. If any step has `status == "failed"`, the feed entry status SHALL be set to `FeedFailed` rather than `FeedDone`. Only if no step has `status == "failed"` SHALL the entry be set to `FeedDone`.

This ensures the signal board LED, the activity feed badge (`✓ done` / `✗ failed`), and the filter counts all reflect the true outcome of a pipeline run.

#### Scenario: All steps done — entry becomes FeedDone
- **WHEN** a `jobDoneMsg` arrives and all steps have `status == "done"`
- **THEN** the feed entry status is set to `FeedDone` (green ✓ done)

#### Scenario: Any step failed — entry becomes FeedFailed
- **WHEN** a `jobDoneMsg` arrives and at least one step has `status == "failed"`
- **THEN** the feed entry status is set to `FeedFailed` (red ✗ failed) regardless of the process exit code

#### Scenario: No steps recorded — entry becomes FeedDone
- **WHEN** a `jobDoneMsg` arrives and the feed entry has no step records
- **THEN** the feed entry status is set to `FeedDone` (existing behavior preserved)

#### Scenario: Signal board shows failed for partially-failed run
- **WHEN** a pipeline exits 0 but one step reported `status:failed`
- **THEN** the signal board row displays the red LED and `failed` status label
