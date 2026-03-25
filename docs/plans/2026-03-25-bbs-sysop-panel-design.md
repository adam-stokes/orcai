# BBS-Style Sysop Panel Redesign

**Date:** 2026-03-25
**Status:** Approved

## Overview

Redesign the sysop panel (`internal/sidebar`, `cmd/orcai-sysop`) to match the aesthetic of classic BBS systems (inspired by Renegade BBS), while adapting the layout to display live agent telemetry for the ABS (Agentic Bulletin System).

## Behavior Changes

### Popup instead of split-pane

`RunToggle()` currently splits the current tmux window into a 30% side-pane and tracks
visibility state via a marker file. This is replaced with a simple `display-popup` call —
identical to how the picker works.

- `^spc t` opens `orcai-sysop` in a `display-popup -E -w 120 -h 40` popup
- `q` or `ctrl+c` closes it
- The `Run()` fullscreen path (direct invocation) is retained unchanged

**Removed:** `isPanelVisible`, `setPanelVisible`, `panelVisiblePath`, and the
`~/.config/orcai/.panel-N` marker files.

## Layout (120×40)

```
┌──────────────────────────────────────────────────────────────────────────────────────┐
│                               ABS · SYSOP MONITOR                                    │
└──────────────────────────────────────────────────────────────────────────────────────┘

┌─── Active Nodes ───────────────┐  ┌─── Node Details ───────────────┐  ┌─── Activity Log ───────────┐
│ [1] claude-opus-4.6   [BUSY]   │  │ Provider ........ Claude        │  │ 12:45  NODE01  done $0.045 │
│ [2] opencode          [IDLE]   │  │ Model ........... opus-4.6      │  │ 12:44  NODE02  streaming   │
│ [3] shell             [WAIT]   │  │ Status .......... streaming      │  │ 12:43  NODE01  done $0.032 │
│                                │  │ Tokens .......... 12k↑ / 3k↓   │  │                            │
│                                │  │ Cost ............ $0.045        │  │                            │
└────────────────────────────────┘  └────────────────────────────────┘  └────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────────────────────┐
│          [enter] focus node          [x] kill node          [↑↓] navigate           │
└──────────────────────────────────────────────────────────────────────────────────────┘
NODES: 3 ACTIVE  │  STREAMING  │  CLAUDE  │  $0.157 TOTAL  │  12:48
```

Three-column layout, all columns equal height (tallest column sets height). Below the
columns: a full-width actions bar and an unboxed status strip.

## Color Scheme

All colors are from the ABS/Dracula palette:

| Element | ANSI |
|---------|------|
| Box borders (`│ ─ ┌ ┐ └ ┘ ├ ┤`) | `\x1b[36m` cyan |
| Section headers (`─── Name ───`) | `\x1b[96m` bright cyan |
| Key labels (`[1]`, `[enter]`) | `\x1b[96m` bright cyan |
| `[BUSY]` badge | `\x1b[92m` bright green |
| `[WAIT]` badge | `\x1b[93m` bright yellow |
| `[IDLE]` badge | dim teal (`\x1b[38;5;66m`) |
| Selected row | `\x1b[48;5;235m` bg + `\x1b[97m` bright white |
| Status strip text | `\x1b[36m` cyan |
| Status strip separators | dim teal |

## What Stays the Same

- All model fields (`windows`, `cursor`, `sessions`, `log`)
- `Update()` logic: window polling every 3s, telemetry handling, cursor nav, `enter` focus, `x` kill
- `ParseWindows()` and `NewWithWindows()` (used in tests)
- `Run()` entry point

## Files Changed

| File | Change |
|------|--------|
| `internal/sidebar/sidebar.go` | Replace ANSI constants + all view helpers + `View()` + `RunToggle()`, delete panel visibility helpers |
| `internal/bootstrap/bootstrap.go` | Chord `t` already calls `orcai-sysop toggle`; `RunToggle` now uses `display-popup` so no conf change needed |
