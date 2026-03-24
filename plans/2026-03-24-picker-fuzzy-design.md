# Picker Fuzzy Search & Unified Session Launcher Design

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Date:** 2026-03-24
**Status:** Approved

## Overview

Replace the current wizard-style picker (provider → model → workdir) with a single-screen fuzzy picker. Everything that can start a session — existing sessions, pipelines, skills, agents, and raw providers — appears in one grouped list. Typing filters across all groups simultaneously using `sahilm/fuzzy`. Group headers hide when a search leaves them empty.

Not all plugins appear in the picker. Only session-starters are shown. Display-only plugins (sidebars, status widgets) are excluded; for now, only `TypePipeline` entries from discovery are added since `TypeNative` and `TypeCLIWrapper` entries overlap with the existing providers list. A future `SessionStarter bool` on `discovery.Plugin` will generalize this.

---

## Item Model

Every row is a `PickerItem`:

```go
type PickerItem struct {
    Kind        string // "session" | "pipeline" | "skill" | "agent" | "provider"
    Name        string
    Description string
    Source      string // "[global]" "[project]" "[copilot]" — empty for providers/sessions
    // launch metadata
    ProviderID  string // for kind=provider; also set after skill/agent picks a provider
    ModelID     string
    InjectText  string // entry.Inject from IndexEntry (e.g. "/golang-patterns", "@beast-mode ")
    PipelineFile string // for kind=pipeline
    SessionIndex string // for kind=session — tmux window index
    // fuzzy match state
    matchIndexes []int
}

func (p PickerItem) Filter() string { return p.Name + " " + p.Description }
func (p *PickerItem) SetMatch(m fuzzy.Match) { p.matchIndexes = m.MatchedIndexes }
```

---

## Groups & Ordering

Groups render in this order. Headers are omitted when fuzzy filtering leaves the group empty.

```
── sessions ──       existing tmux windows (resume, no new launch)
── pipelines ──      *.pipeline.yaml from ~/.config/orcai/pipelines/
── skills ──         from chatui.ScanIndex — ~/.claude/skills/, project skills
── agents ──         from chatui.ScanIndex — ~/.claude/commands/, ~/.copilot/agents/
── providers ──      claude, copilot, opencode, ollama, shell (same as today)
```

The `Filter()` string for each item is `name + " " + description`, so typing "research" finds a pipeline named `my-research-pipeline`, and typing "beast" finds the `beast-mode` agent.

---

## Selection Flows

| Selected Kind | Next Steps |
|---------------|------------|
| session       | `focusWindow(index)` → close picker |
| pipeline      | workdir screen → `pipeline run <name>` in new tmux window |
| skill / agent | provider screen → workdir screen → launch CLI → `opsx.ProviderSend(injectText, providerID, dir)` |
| provider      | model screen (if models exist) → workdir screen → launch (same as today) |

The **provider screen** (shown after picking a skill/agent) lists only installed CLIs from `buildProviders()`. Copilot-specific agents (`Source: "cli:copilot"`) pre-select Copilot in that screen but the user can change it.

---

## Fuzzy Matching

- Add `github.com/sahilm/fuzzy` to `go.mod`.
- The search input sits above the list, always focused.
- On each keystroke: run `fuzzy.FindFrom(query, source)` over all non-header items, collect matches sorted by score, re-render with match indexes highlighted.
- Matched characters: pink (`Color("212")`). Unmatched name: normal. Description + source tag: dim (`Color("240")`).
- Empty query → show all items in group order (no scoring).

---

## Rendering

Popup stays at its current `42×14` tmux `display-popup` size. Headers count as one row each. Items are scrollable. Layout per row:

```
▎ beast-mode          top-notch coding agent   [global]
  golang-patterns     idiomatic Go patterns    [global]
```

Active item: pink on dark selection background (`Color("236")`).
Source tag right-aligned or inline dim — whichever fits at 42 cols.

---

## Picker States

```
StateSearch   — main fuzzy list (replaces StateProvider)
StateModel    — model picker for providers with multiple models (unchanged)
StateProvider — provider picker when launching a skill/agent (new use)
StateWorkdir  — working directory input (unchanged)
StateWorkflow — openspec workflow choice (unchanged)
StateOpenSpecName — openspec feature name input (unchanged)
```

`StateSearch` replaces `StateProvider` as the initial state. The existing `StateModel`, `StateWorkdir`, `StateWorkflow`, and `StateOpenSpecName` screens are reused unchanged.

---

## New Packages / Files Changed

| File | Change |
|------|--------|
| `internal/picker/picker.go` | Replace `StateProvider` list with `StateSearch` fuzzy list; add `PickerItem`, fuzzy filter logic, group rendering |
| `internal/picker/items.go` | New file: `buildPickerItems()` — calls `ScanIndex`, `ScanPrompts`, `discovery.Discover`, `listExistingSessions`, `buildProviders` and returns `[]PickerItem` grouped |
| `go.mod` / `go.sum` | Add `github.com/sahilm/fuzzy` |

The indexer (`internal/chatui/indexer.go`) and discovery (`internal/discovery/discovery.go`) are read-only from the picker's perspective — no changes needed there.

---

## What Does Not Change

- `internal/sidebar/sidebar.go` — sidebar unchanged; `[n]` still opens the picker popup
- `internal/chatui/` — indexer and sessions registry unchanged
- `internal/discovery/` — unchanged
- Pipeline launch uses the existing `pipeline run` subcommand via tmux `new-window`
- Workdir, workflow, and OpenSpec screens are pixel-identical to today
