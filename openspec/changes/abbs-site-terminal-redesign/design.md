## Context

The current ORCAI ABS site is a multi-page Astro static site. Each "page" (plugins, pipelines, changelog, etc.) is a separate `.astro` file that causes a full browser navigation and page reload when visited. The `bbs.js` file has rudimentary keyboard shortcuts that `window.location.href` to other pages — contradicting the terminal-emulator identity of the product.

The site already has strong aesthetic foundations: hex-dump canvas background, Dracula palette, monospace fonts, ANSI logo, typewriter effect, and box-drawing CSS borders. The redesign builds on these rather than replacing them.

The product has also been rebranded from "ABS" to "ABBS — Agentic Bulletin Board System" and refactored so that the TUI itself does not scroll pages — only individual widgets and shell panes scroll. The website should embody this same model.

## Goals / Non-Goals

**Goals:**
- Single-page terminal emulator: all "navigation" swaps the active screen in-place with no page reload
- Full keyboard navigation: every screen reachable by keyboard shortcut; mouse optional
- Viewport never scrolls — only the active content pane scrolls internally (overflow-y: auto on the pane, hidden on body)
- ANSI box-drawing art as the primary layout and decoration primitive (borders, banners, status bars, dividers)
- Marketing copy updated to reflect ABBS rebrand, pipeline customization focus, and hacker-customizable agent workspace identity
- Pipeline showcase screen with live-typed YAML demo and ANSI step diagram

**Non-Goals:**
- Replacing Astro as the build tool — it remains the static site generator
- Server-side rendering or dynamic backends
- Mobile/responsive breakpoints — this is a desktop-first terminal aesthetic
- Accessibility overhaul (though keyboard nav improves baseline a11y)
- Rewriting the Go/BubbleTea TUI itself

## Decisions

### Decision: Single-page shell with JS screen router (not Astro file-per-page)

All screens are defined as `<section data-screen="...">` elements inside a single `index.astro`. The JS screen router shows/hides screens, updates the nav indicator, and dispatches keyboard events to the active screen — no network request, no scroll-to-top jank.

**Alternatives considered:**
- Astro view transitions API: cleaner but still triggers navigation events and is browser-dependent; inconsistent with "no reload" feel
- iframes per screen: too heavy; breaks CSS variable inheritance and keyboard focus
- Hash routing (SPA): would work but adds URL complexity and back-button behavior we don't want

### Decision: ANSI art rendered as `<pre>` blocks with CSS color variables

Box-drawing characters (`╔ ═ ╗ ║ ╠ ╣ ╚ ╝ ├ ┤`) and block characters (`▓ ░ █`) are used directly in HTML `<pre>` elements. Color is applied via CSS classes (`fg-purple`, `fg-cyan`, `fg-green`, `fg-comment`) that map to Dracula variables. This is maintainable, SSR-friendly, and avoids canvas overhead for static art.

**Alternatives considered:**
- Canvas-rendered ANSI: powerful but not SSR-renderable; harder to update copy
- SVG art: not authentically terminal-aesthetic
- CSS borders only: already done; this goes further with actual box-drawing glyphs

### Decision: Content panes use `overflow-y: auto` with fixed height; `body` is `overflow: hidden`

Each screen's `.term-pane` fills the viewport minus the header and status bar (using CSS `calc` and `vh`). Only the pane scrolls. The `<body>` never scrolls. This matches how orcai widgets behave in the TUI.

### Decision: Keyboard dispatcher is a global singleton in `bbs.js`

A `KeyboardRouter` object:
1. Maintains the current active screen ID
2. Dispatches global shortcuts (1-7 for screens, `?` for help, `q` for "disconnect" fade)
3. Delegates screen-local shortcuts to per-screen handler maps
4. Ignores keys when focus is inside `<input>` or `<textarea>`

This keeps keyboard logic centralized and prevents duplicate listeners accumulating on screen switches.

### Decision: Pipeline showcase uses typed YAML with cursor, not a video or gif

The pipeline screen plays back a hardcoded YAML pipeline definition character-by-character using a JS typewriter adapted from the existing `typewriter()` function. This is pure JS, loads instantly, and reinforces the "code-first" product identity. A separate ANSI step-diagram rendered in `<pre>` shows the pipeline DAG visually.

### Decision: Astro pages collapsed to `index.astro` only; other `.astro` files become components

`plugins.astro`, `pipelines.astro`, etc. are converted to Astro components (`PluginsScreen.astro`, etc.) imported into `index.astro`. The Astro router still functions, but all content lives in one output HTML file.

## Risks / Trade-offs

- **SEO regression**: Single-page means all content shares one URL. Search engines won't index `/plugins` or `/pipelines` separately. → Mitigation: add `<meta>` og tags and a sitemap pointing to anchors; the audience (CLI developers) skews toward direct links and GitHub anyway.
- **JS-heavy navigation**: If JS fails to load, the page shows all screens stacked. → Mitigation: CSS `.screen` default is `display:none` except the first; a `<noscript>` banner advises enabling JS.
- **Large initial HTML**: All screen content is inline. → Not a concern for a static marketing site; total HTML will be <100KB.
- **Keyboard shortcut conflicts**: Browser shortcuts (e.g., `Ctrl+P` for print) may conflict. → We use single-key shortcuts without modifiers for screen switching; only safe unmodified keys.

## Migration Plan

1. Build new `index.astro` as the single shell, importing all screen components
2. Rewrite `bbs.js`: replace per-page keyboard nav with `KeyboardRouter`; add screen switcher
3. Rewrite `bbs.css`: add `.term-pane`, `.screen`, `.ansi-border`, `.status-bar` primitives; lock `body` scroll
4. Delete `plugins.astro`, `pipelines.astro`, `getting-started.astro`, `changelog.astro`, `themes.astro` after content migrated to components
5. Update `astro.config.mjs` if needed to reflect single entry point
6. Deploy to GitHub Pages as before (no infra change)

**Rollback**: git revert to prior commit; site is static so no migration state to undo.

## Open Questions

- Should the pipeline YAML typewriter replay on every screen visit or only the first? (Preference: every visit, for demo effect)
- Themes screen: currently minimal — worth expanding to a live palette switcher? (Out of scope for this change; noted for follow-up)
