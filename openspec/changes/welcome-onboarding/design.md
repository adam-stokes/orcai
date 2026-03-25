## Context

`internal/welcome/welcome.go` is a BubbleTea model. The `Update` function currently handles `tea.KeyMsg` with a single block: if the key is `enter`, set `launchPicker = true`, then unconditionally call `tea.Quit` for any key. This means any keypress (spacebar, arrow, letter) immediately exits the welcome screen — contradicting the `welcome-dashboard` spec requirement for a persistent dashboard.

`buildHelp` returns a static string showing chord hints. The last line reads `"  ── enter new session · any key continue ──"` which actively advertises the undesired any-key-exit behavior. With `sensible-window-defaults` merged, the chord table now includes `c` (new window), `|`/`-` (splits), and arrow keys (pane nav), none of which appear in the current help text.

## Goals / Non-Goals

**Goals:**
- Fix `Update` so only `q`, `esc`, `ctrl+c` quit; `enter` opens picker; all other keys are ignored
- Refresh `buildHelp` with the complete current chord reference and a proper close hint
- Keep changes minimal and scoped to `welcome.go`

**Non-Goals:**
- Interactive navigation (no cursor, no selected items) — the welcome screen is read-only with a few hotkeys
- Changing the banner (`buildWelcomeArt`) or bus/telemetry logic
- Adding new keybindings beyond the existing chord table

## Decisions

### 1. Key handling in Update

**Decision:** Explicit allowlist — switch on the key string, handle `q`/`esc`/`ctrl+c` as quit, `enter` as picker launch + quit, and fall through (return without quitting) for everything else.

```go
case tea.KeyMsg:
    switch msg.String() {
    case "q", "esc", "ctrl+c":
        return m, tea.Quit
    case "enter":
        m.launchPicker = true
        return m, tea.Quit
    }
```

**Alternative considered:** Denylist (quit on everything except a few keys). Rejected — same surface area, harder to reason about.

### 2. Help text structure

Replace the footer line `"  ── enter new session · any key continue ──"` with two lines:

```
  enter  new session   (pick provider + model)
  q / esc  close
```

Add the new chord entries after the existing ones:

```
    c  new shell window
    |  split right     -  split down
    ←→↑↓  navigate panes
```

Keep the existing entries (`n`, `t`, `p`, `q`, `d`) as-is.

## Risks / Trade-offs

- **Behavior regression**: Any test or user workflow that relied on "press any key to advance" will break. That behavior was already contradicted by the existing spec, so this is a spec-compliance fix, not a regression.
- **Footer length**: Adding more chord entries increases help text height. On very short terminals the footer may scroll off. Acceptable given typical terminal sizes.

## Migration Plan

Bootstrap generates `tmux.conf` on every launch. The welcome widget is invoked via `orcai _welcome` in the initial new-session command. No migration needed — the change takes effect on next launch.

## Open Questions

*(none)*
