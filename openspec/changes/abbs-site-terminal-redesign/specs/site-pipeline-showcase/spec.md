## ADDED Requirements

### Requirement: Pipeline screen displays a typed YAML pipeline demo
The Pipelines screen SHALL include a `<pre class="pipeline-demo">` element that plays back a hardcoded YAML pipeline definition character-by-character using the existing `typewriter()` engine. Playback SHALL start automatically when the Pipelines screen becomes active and restart on each subsequent visit.

#### Scenario: YAML typewriter starts on screen activation
- **WHEN** the user navigates to the Pipelines screen (key `4` or nav click)
- **THEN** the pipeline demo pre-element begins typing out the YAML from the beginning

#### Scenario: YAML typewriter restarts on re-visit
- **WHEN** the user navigates away from Pipelines and returns
- **THEN** the YAML demo resets and begins typing again from the start

#### Scenario: YAML content represents a real pipeline structure
- **WHEN** the YAML demo completes
- **THEN** the displayed YAML contains `steps:`, at least one `provider:` entry, at least one `plugin:` entry, and `input:`/`output:` type declarations

### Requirement: Pipeline screen displays an ANSI DAG diagram
Adjacent to the YAML demo, the Pipelines screen SHALL display a static ANSI art DAG diagram rendered in a `<pre>` element. The diagram SHALL show at minimum: a Provider node, a Plugin node, and an Output node connected by arrows using `→` or `▶` and `╌` characters. Node boxes SHALL use box-drawing characters.

#### Scenario: DAG diagram visible alongside YAML demo
- **WHEN** the Pipelines screen is active
- **THEN** both the YAML typewriter and the ANSI DAG diagram are visible in the pane (side by side or stacked)

#### Scenario: DAG diagram uses box-drawing nodes
- **WHEN** the DAG diagram is rendered
- **THEN** each node is enclosed in a box-drawing border and labeled with its role (e.g., `PROVIDER`, `PLUGIN`, `OUTPUT`)

### Requirement: Pipeline screen has hacker-style marketing copy
The Pipelines screen SHALL display marketing copy framed as terminal output: prefixed with `>` prompt characters, using Dracula color classes, and describing the pipeline system in hacker/operator voice. Copy SHALL mention: YAML-first design, typed input/output schemas, composability with any CLI tool, and agent orchestration.

#### Scenario: Copy uses terminal prompt style
- **WHEN** the Pipelines screen is rendered
- **THEN** marketing paragraphs are prefixed with `>` or displayed in a `<pre>` with prompt-style formatting

#### Scenario: Copy mentions typed schemas
- **WHEN** the Pipelines screen content is visible
- **THEN** the copy references typed `input:` and `output:` schemas as a core feature

### Requirement: Pipeline screen keyboard shortcut `r` replays the demo
When the Pipelines screen is active, pressing `r` SHALL reset and replay the YAML typewriter demo from the beginning.

#### Scenario: `r` replays YAML demo
- **WHEN** the Pipelines screen is active and the user presses `r`
- **THEN** the pipeline demo pre-element clears and begins typing the YAML from scratch
