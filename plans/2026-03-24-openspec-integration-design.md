# OpenSpec Integration Design

**Date:** 2026-03-24
**Status:** Approved

## Overview

Integrate [openspec.dev](https://openspec.dev/) into orcai on two parallel tracks:

1. **Dev workflow (now):** Use openspec as the authoritative planning layer for orcai's own development — proposals drive the feature roadmap.
2. **Plugin integration (later):** Expose openspec as a pipeline step in the prompt builder so users can generate structured feature proposals from within the TUI.

---

## Track 1 — Dev Workflow

### Setup

```bash
npm install -g @fission-ai/openspec@latest
cd /path/to/orcai
openspec init
```

This creates:
- `.openspec/` — project config and auto-generated AI agent guidance files (Claude Code slash commands wired automatically)
- `openspec/changes/` — one folder per proposal, each containing `proposal.md`, `specs/`, `design.md`, `tasks.md`
- `openspec/changes/archive/` — completed work moves here

### Day-to-Day Workflow

1. `/opsx:propose <description>` — describe a feature or change; openspec generates a full structured proposal
2. Review `proposal.md` + `design.md` before any code is written
3. `/opsx:apply` — Claude Code executes the generated `tasks.md` checklist
4. `/opsx:archive` — moves completed change to archive

### Relationship to Existing `docs/plans/`

`docs/plans/` holds architecture and design documents (plugin system design, prompt builder design, etc.) — unchanged.

Going forward, `openspec/changes/` holds feature proposals and task checklists. The brainstorming → writing-plans workflow feeds into `/opsx:propose`: brainstorm → approve design → generate openspec proposal → apply.

---

## Track 2 — Plugin Integration

### Phase 1 — Tier 2 CLI Wrapper

Once `CliAdapter` is built (part of the plugin system implementation), add a sidecar YAML:

**`~/.config/orcai/wrappers/openspec.yaml`**
```yaml
name: openspec
description: Generate structured feature proposals from a description
input_schema:
  type: string
  description: Feature or change description
output_schema:
  type: string
  description: Markdown proposal (proposal.md contents)
command: openspec
args: ["propose", "--stdout"]
```

openspec auto-appears in the prompt builder's provider picker. A pipeline step using it:

```yaml
- id: spec
  plugin: openspec
  input: "{{step1.out}}"
```

User types a feature description in the prompt builder → openspec runs → structured proposal streams back into the TUI.

### Phase 2 — Optional Tier 1 Native Plugin

Promote to a Go binary (`orcai-plugin-openspec`) implementing `Execute` + `Capabilities` gRPC only if richer needs emerge:
- Streaming line-by-line output
- Event bus publishing to `pipeline.<name>.step.spec.out`
- Bidirectional control

Not needed until Tier 2 proves insufficient.

---

## Directory Structure

```
orcai/
├── .openspec/                    # openspec config + Claude Code guidance
├── openspec/
│   └── changes/
│       ├── active/               # in-flight proposals
│       │   └── YYYY-MM-DD-<feature>/
│       │       ├── proposal.md
│       │       ├── design.md
│       │       ├── tasks.md
│       │       └── specs/
│       └── archive/              # completed work
├── docs/
│   └── plans/                    # existing architecture/design docs, unchanged
└── ~/.config/orcai/
    └── wrappers/
        └── openspec.yaml         # Tier 2 sidecar (added when CliAdapter ships)
```

---

## What Does Not Change

- `docs/plans/` — architecture and design docs remain here, authored via brainstorming workflow
- Existing plugin system design — openspec plugin follows the Tier 2 → Tier 1 promotion path already designed
- Prompt builder UX — openspec appears as a provider in the existing picker, no new UI needed
