## ADDED Requirements

### Requirement: Pipeline tmux window remains interactive after run completion
After a pipeline finishes (success or failure), the tmux window SHALL transition to an interactive shell (`$SHELL`) rather than showing a dead pane. The shell SHALL inherit the pipeline's working directory and environment.

#### Scenario: Successful pipeline leaves live shell
- **WHEN** a pipeline run exits with code 0
- **THEN** the tmux window shows an interactive shell prompt and the pane is NOT dead

#### Scenario: Failed pipeline leaves live shell
- **WHEN** a pipeline run exits with a non-zero code
- **THEN** the tmux window shows an interactive shell prompt so the user can inspect the environment

#### Scenario: User can exit the shell manually
- **WHEN** the user types `exit` or presses Ctrl-D in the post-run shell
- **THEN** the pane closes normally (remain-on-exit fires as a safety net)
