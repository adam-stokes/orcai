## Why

The ORCAI ABS website uses traditional multi-page navigation with full page refreshes, contradicting the product's core identity as a terminal-native, no-browser tool. The site needs to behave like the TUI it markets: a single-screen terminal emulator with widget-level scrolling, full keyboard navigation, and ANSI art as the primary visual language — not a conventional marketing site dressed in dark colors.

## What Changes

- Replace multi-page Astro routing with a single-page terminal emulator shell that swaps "screens" without page refresh
- Navigation becomes a BBS-style menu bar driven entirely by keyboard shortcuts (no mouse required)
- Content areas become scrollable terminal "panes" — only the widget/pane scrolls, never the viewport
- ANSI art (box-drawing, color blocks, ASCII banners) becomes the primary layout primitive, not CSS cards
- Marketing copy updated to reflect the ABBS rebrand (Agentic Bulletin Board System), pipeline customization, and plugin architecture refactors
- Keyboard shortcut overlay / help screen accessible via `?` or `F1`
- "Screens" replace pages: HOME, ABOUT, GETTING STARTED, PLUGINS, PIPELINES, CHANGELOG, THEMES

## Capabilities

### New Capabilities

- `site-terminal-shell`: Single-page terminal emulator shell — full keyboard nav, screen switching without page reload, viewport never scrolls
- `site-ansi-layout`: ANSI/box-drawing art as the primary layout system — borders, banners, status bars, pane headers all rendered in ASCII art
- `site-pipeline-showcase`: Dedicated pipeline marketing screen with live-typed YAML examples, step diagrams in ANSI art, hacker-style "run pipeline" demo

### Modified Capabilities

- `welcome-dashboard`: Update content and copy to reflect ABBS rebrand and new plugin/pipeline architecture; widget descriptions to match current TUI behavior (widgets scroll, not pages)

## Impact

- `site/src/layouts/` — Base and BBS layouts replaced or heavily modified; single-page shell layout introduced
- `site/src/components/` — Nav replaced with keyboard-driven screen switcher; new TerminalPane, AnsiBorder, StatusBar components
- `site/public/js/bbs.js` — Major rewrite: screen router, keyboard dispatcher, pane scroll manager
- `site/public/css/bbs.css` — Layout overhaul: full-viewport pane model, no scrolling body
- `site/src/pages/` — All pages become screen definitions loaded into the shell (or collapsed to index.astro)
- No backend changes; purely static site (Astro remains the build tool)
