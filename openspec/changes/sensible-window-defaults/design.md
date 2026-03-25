## Context

Orcai manages a tmux session and uses `ctrl+space` as a chord leader for all orcai-specific actions. The current `buildTmuxConf` function in `bootstrap.go` fully suppresses tmux's built-in window status display (`window-status-format ""`) and binds no keys for window/pane creation, navigation, or destruction. Users arriving from standard tmux have no keyboard path to these operations.

The `keybindings.go` `actionMap` provides a user-facing override layer; the hardcoded chords in `buildTmuxConf` provide the zero-config defaults.

## Goals / Non-Goals

**Goals:**
- Bind sensible window management chords under the existing `ctrl+space` leader (no new leader)
- Bind sensible pane split and navigation chords under the same leader
- Show a minimal Dracula-themed window indicator in the status bar
- Expose all new operations as named actions in `actionMap` so `keybindings.yaml` can override them

**Non-Goals:**
- A separate window management TUI or picker UI
- Session management beyond what already exists (`n` → session picker)
- Renaming windows via keyboard (can be done directly in tmux if needed)
- Any changes to pane layout config (`layout.yaml`) or the existing `widgetdispatch` layer

## Decisions

### 1. Reuse the existing `ctrl+space` chord leader

**Decision:** All new bindings live under the `orcai-chord` table, not a sub-table or a new prefix.

**Rationale:** A single leader is simpler and consistent with the existing mental model. The chord table has enough unused keys. Adding a sub-table (e.g. `ctrl+space w` for window ops) would add latency and cognitive load for a small number of operations.

**Alternative considered:** Standard tmux `ctrl+b` prefix in addition to `ctrl+space`. Rejected — orcai's config does not set a prefix key and we don't want to reintroduce one.

### 2. Key assignments

| Chord | Operation | Rationale |
|-------|-----------|-----------|
| `c` | New plain window | `ctrl+b c` is the canonical tmux binding; muscle memory |
| `[` | Previous window | `ctrl+b p` is taken; `[` / `]` are common alternatives |
| `]` | Next window | Symmetric with `[` |
| `\|` | Split pane vertically (side-by-side) | Visual metaphor |
| `-` | Split pane horizontally (top/bottom) | Visual metaphor |
| Arrow keys | Navigate panes | Natural, no mnemonic required |
| `x` | Kill current pane | `ctrl+b x` is canonical |

**`&` (kill window) is omitted** — users can close a window by killing all its panes (`x` repeatedly) or via `q` → quit flow. Adding `&` risks accidental window loss; the cost is low.

### 3. Status bar window indicator

**Decision:** Restore `window-status-format` and `window-status-current-format` with Dracula palette values. Place the window list in the status-left area (after the ORCAI label) rather than center, keeping the right side for hints and clock.

**Rationale:** Status-left already has spare width. Centering the window list requires setting `status-justify center` which affects layout globally and may look odd with only one window.

**Format:**
- Inactive: `#[fg=#6272a4] #I:#W ` (muted purple, index:name)
- Active: `#[fg=#f8f8f2,bold] #I:#W ` (foreground white, bold)

### 4. `actionMap` entries

New entries mirror the chord bindings so `keybindings.yaml` can remap them:

```
"new-window"        → ["new-window"]
"prev-window"       → ["previous-window"]
"next-window"       → ["next-window"]
"split-pane-right"  → ["split-window", "-h"]
"split-pane-down"   → ["split-window", "-v"]
"kill-pane"         → ["kill-pane"]
"select-pane-left"  → ["select-pane", "-L"]
"select-pane-right" → ["select-pane", "-R"]
"select-pane-up"    → ["select-pane", "-U"]
"select-pane-down"  → ["select-pane", "-D"]
```

## Risks / Trade-offs

- **Chord key conflicts** → Mitigation: Verified all assigned keys (`c`, `[`, `]`, `|`, `-`, arrows, `x`) are currently unbound in `orcai-chord`. If future actions need these, the `keybindings.yaml` override layer handles user-specific remapping.
- **Status bar length** → The window list grows with window count; very long window names may push hints off-screen. Mitigation: truncate window names at 12 chars via `#W` with `window-status-format` length limit (`#{=/12/…:window_name}`). Low priority for now.
- **Mouse already on** → Mouse click to switch windows already works; these bindings add keyboard parity only. No conflict.

## Migration Plan

Bootstrap regenerates `tmux.conf` on every `orcai` invocation — users get the new bindings and status bar automatically on next launch. No migration steps required.

## Open Questions

- Should `c` create a window with a default name (e.g. `shell`) or use tmux's unnamed default? Unnamed is simpler and avoids stale names. Defaulting to unnamed for now.
