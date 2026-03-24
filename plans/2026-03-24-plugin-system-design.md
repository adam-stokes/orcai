# Plugin System & Prompt Builder Design

**Date:** 2026-03-24
**Status:** Approved

## Overview

Transform orcai into a universal plugin host where every CLI tool — AI or otherwise — is a plugin, and pipelines of plugins are themselves first-class discoverable plugins. A charm BBS-style prompt builder lets users compose, conditionally branch, and save these pipelines interactively.

---

## Architecture

Everything in orcai becomes a plugin. Three layers:

```
┌─────────────────────────────────────────────────┐
│              Prompt Builder (TUI)               │
│         80% modal · bubbletea · lipgloss        │
└───────────────────┬─────────────────────────────┘
                    │ writes .pipeline.yaml
┌───────────────────▼─────────────────────────────┐
│             Plugin Registry                     │
│   discovery/ scans: native plugins + CLI        │
│   wrappers + compiled pipeline plugins          │
└──────┬────────────┬────────────┬────────────────┘
       │            │            │
┌──────▼───┐  ┌─────▼────┐  ┌───▼──────────────┐
│ Tier 1   │  │ Tier 2   │  │ Tier 3           │
│ Native   │  │ CLI      │  │ Pipeline Plugins  │
│ go-plugin│  │ Wrappers │  │ (.pipeline.yaml   │
│ binaries │  │ (stdin/  │  │  interpreted at   │
│          │  │  stdout) │  │  runtime)         │
└──────────┘  └──────────┘  └───────────────────┘
                    │
        ┌───────────▼───────────┐
        │     Event Bus (gRPC)  │
        │  pub/sub · topics ·   │
        │  plugin-to-plugin msg │
        └───────────────────────┘
```

Every plugin registers on the event bus at startup via `StartRequest.bus_address`. Pipelines are plugins that orchestrate other plugins by publishing/subscribing to typed bus topics.

---

## Plugin System

### Tier 1 — Native go-plugins (full capability)

- Implement the `OrcaiPlugin` gRPC service with two new methods: `Execute` (streaming I/O) and `Capabilities` (self-describing schema)
- Binaries live in `~/.config/orcai/plugins/` — scanned by `discovery.go`
- Examples: `orcai-plugin-openspec`, `orcai-plugin-openclaw`, `orcai-plugin-claude-adapter`
- Full bidirectional event bus access

### Tier 2 — CLI Wrappers (auto-adapted, zero changes required)

- Any CLI in PATH is automatically wrapped: spawn subprocess, communicate via stdin/stdout/JSON envelope
- A `CliAdapter` struct implements the same internal `Plugin` interface
- Optional sidecar YAML (`~/.config/orcai/wrappers/<name>.yaml`) declares input/output schema
- Can be promoted to Tier 1 by writing a native plugin binary

### Tier 3 — Pipeline Plugins

- `.pipeline.yaml` files in `~/.config/orcai/pipelines/` are loaded by the pipeline interpreter at runtime
- Appear in discovery as first-class plugins with name, version, and capabilities
- Can be referenced by other pipelines — pipelines compose
- Saved pipelines show up in the prompt builder's plugin picker

---

## Proto Extensions

Add to `plugin.proto`:

```protobuf
message ExecuteRequest  { string input = 1; map<string,string> vars = 2; }
message ExecuteResponse { string chunk = 1; bool done = 2; string error = 3; }
message CapabilityList  { repeated Capability items = 1; }
message Capability      { string name = 1; string input_schema = 2; string output_schema = 3; }

// Added to OrcaiPlugin service:
rpc Execute(ExecuteRequest)  returns (stream ExecuteResponse);
rpc Capabilities(Empty)      returns (CapabilityList);
```

---

## Pipeline YAML Format

```yaml
name: my-research-pipeline
version: "1.0"
steps:
  - id: step1
    type: input
    prompt: "Enter your research topic:"

  - id: step2
    plugin: claude
    model: claude-sonnet-4-6
    prompt: "Summarize this topic: {{step1.out}}"
    condition:
      if: "contains:spec"
      then: step3a
      else: step3b

  - id: step3a
    plugin: openspec
    input: "{{step2.out}}"

  - id: step3b
    plugin: openclaw
    input: "{{step2.out}}"

  - id: output
    type: output
    publish_to: "pipeline.my-research-pipeline.done"
```

**Template variables:** `{{stepN.out}}` interpolates prior step output into prompts and inputs.

**Condition expressions:** `contains:<str>`, `matches:<regex>`, `len > <n>`, `always` — no scripting language.

---

## Data Flow

```
pipeline.start         → bus topic: pipeline.<name>.start
step N executes        → plugin receives input via Execute() gRPC call
plugin streams output  → publishes to: pipeline.<name>.step.<N>.out
condition evaluated    → pipeline interpreter routes to next step
step N+1 executes      → receives {{stepN.out}} interpolated into prompt
pipeline.done          → publishes to: pipeline.<name>.done (payload = final output)
```

Other plugins or pipelines subscribe to `pipeline.<name>.done` to chain without the builder needing to know about downstream consumers.

---

## Prompt Builder UX

80% modal overlay on the active view. Pure bubbletea, charm BBS aesthetic — bordered panels, ANSI color.

```
╔══════════════════════════════════════════════════════════════╗
║  PIPELINE BUILDER                              [?] help  [x] ║
╠══════════════════════════════════════════════════════════════╣
║  NAME: my-research-pipeline                                  ║
╠══════════════════╦═══════════════════════════════════════════╣
║  STEPS           ║  STEP 2 — CONFIG                         ║
║ ──────────────── ║  ──────────────────────────────────────── ║
║ [1] input        ║  Provider:  [ claude          ▼ ]        ║
║ [2] claude    ←  ║  Model:     [ claude-sonnet-4 ▼ ]        ║
║ [3] ─ condition  ║  Prompt:    ╔──────────────────────────╗  ║
║     ├ openspec   ║             ║ Summarize: {{step1.out}} ║  ║
║     └ openclaw   ║             ╚──────────────────────────╝  ║
║ [4] output       ║  Condition: if output contains "spec"     ║
║                  ║  → branch:  openspec                      ║
║  [+] add step    ║  → else:    openclaw                      ║
╠══════════════════╩═══════════════════════════════════════════╣
║  [r] run  [s] save  [tab] next field  [↑↓] steps  [esc] quit ║
╚══════════════════════════════════════════════════════════════╝
```

**Left pane:** step list with branch tree visualization, `↑↓` to navigate
**Right pane:** config form for selected step — provider picker, model picker, prompt textarea, condition editor, output routing
**Provider/model pickers:** `bubbles/list` populated live from the plugin registry
**`[r] run`:** executes pipeline inline, streams output back into the modal
**`[s] save`:** writes `~/.config/orcai/pipelines/<name>.pipeline.yaml`, immediately visible in discovery

---

## New Packages

| Path | Purpose |
|------|---------|
| `internal/plugin/` | Universal Plugin interface, Tier 1 host, Tier 2 CliAdapter |
| `internal/pipeline/` | YAML loader, interpreter, condition evaluator, template engine |
| `internal/promptbuilder/` | Bubbletea 80% modal, step list, config form, run view |
| `proto/orcai/v1/plugin.proto` | Add Execute + Capabilities RPC |
| `~/.config/orcai/plugins/` | Native plugin binaries |
| `~/.config/orcai/pipelines/` | Saved pipeline YAML files |
| `~/.config/orcai/wrappers/` | Optional CLI sidecar schema YAMLs |

---

## What Does Not Change

- `internal/bus/` — event bus unchanged
- `internal/discovery/` — extended, not replaced (adds pipeline scanning)
- `internal/chatui/` — unchanged; chatui uses the plugin system but is not rebuilt
- `internal/bridge/` — eventually replaced by `internal/plugin/` Tier 2 adapter, but kept until migration complete
- All existing charmbracelet dependencies — reused throughout
