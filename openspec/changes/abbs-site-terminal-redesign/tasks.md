## 1. CSS Foundation — Terminal Pane Model

- [x] 1.1 Set `body { overflow: hidden; height: 100vh; }` and remove any existing body scroll styles in `bbs.css`
- [x] 1.2 Add `.screen { display: none; }` and `.screen.active { display: flex; flex-direction: column; height: 100%; }` rules
- [x] 1.3 Add `.term-pane { overflow-y: auto; flex: 1; }` with Dracula-palette scrollbar styling (`::-webkit-scrollbar`, `::-webkit-scrollbar-thumb`)
- [x] 1.4 Add `.term-pane-header { position: sticky; top: 0; }` for pane title bars
- [x] 1.5 Add ANSI color utility classes: `fg-purple`, `fg-cyan`, `fg-green`, `fg-yellow`, `fg-pink`, `fg-comment`, `fg-orange` mapped to Dracula CSS variables
- [x] 1.6 Add `.status-bar` fixed bottom bar styles: full width, Dracula background, monospace, flex layout for node/screen/clock/online segments
- [x] 1.7 Add `.help-overlay { position: fixed; inset: 0; z-index: 100; display: none; }` and `.help-overlay.open { display: flex; }` styles

## 2. JS — KeyboardRouter and Screen Switcher

- [x] 2.1 Rewrite `bbs.js`: create `KeyboardRouter` singleton with `activeScreen`, `screens` map, `register(screenId, handlerMap)`, and `dispatch(key)` methods
- [x] 2.2 Implement global key dispatch: keys `1`–`7` call `switchScreen(index)`, `?`/`F1` toggle help overlay, `q`/`Escape` trigger disconnect fade
- [x] 2.3 Implement `switchScreen(id)`: hide all `.screen` elements, show target, update nav `.active` class, update status bar screen name, call active screen's `onEnter()` hook if defined
- [x] 2.4 Add `initClock()` — update `.clock` span every second (already partially exists; ensure it updates the status bar clock element)
- [x] 2.5 Implement help overlay open/close: toggle `.help-overlay.open`, trap focus within overlay while open
- [x] 2.6 Register Pipelines screen local handler: key `r` → reset and replay YAML typewriter demo

## 3. HTML Structure — Single-Page Shell

- [x] 3.1 Refactor `site/src/layouts/Base.astro`: no changes needed to `<head>`; ensure `<body>` has no scroll-enabling styles
- [x] 3.2 Create `site/src/layouts/TerminalShell.astro`: wraps content in `<div class="terminal-shell">` with nav, screens container, status bar, help overlay, and `<slot>`
- [x] 3.3 Rewrite `site/src/components/Nav.astro`: render nav items as keyboard-hinted labels (`[1] HOME  [2] DOCS ...`); remove `href` navigation; add `data-screen` attributes
- [x] 3.4 Rewrite `site/src/pages/index.astro` as the single entry point: import all screen components, render them as `<section data-screen="...">` inside the shell
- [x] 3.5 Add `<noscript>` banner: "Keyboard navigation requires JavaScript. All content is still accessible below."
- [x] 3.6 Add `<div class="status-bar">` at bottom of shell: `NODE-001 | SCREEN: <span class="current-screen">HOME</span> | <span class="clock"></span> | ● ONLINE`

## 4. Screen Components — ANSI Art Layouts

- [x] 4.1 Create `site/src/components/screens/HomeScreen.astro`: hero with `AnsiLogo`, typewriter, `SysinfoBox`, keyboard hint bar using box-drawing decoration
- [x] 4.2 Create `site/src/components/screens/AboutScreen.astro`: pane header `╔══[ ABOUT ]══╗`, ABBS rebrand copy, feature cards reformatted as ANSI box panels
- [x] 4.3 Create `site/src/components/screens/GettingStartedScreen.astro`: pane header, install steps as numbered ANSI panels, code blocks with copy buttons
- [x] 4.4 Create `site/src/components/screens/PluginsScreen.astro`: pane header `╔══[ PLUGINS ]══╗`, plugin architecture overview with ANSI-bordered code example
- [x] 4.5 Create `site/src/components/screens/PipelinesScreen.astro`: two-column layout — left: ANSI DAG diagram `<pre>`; right: `<pre class="pipeline-demo">` for YAML typewriter
- [x] 4.6 Create `site/src/components/screens/ChangelogScreen.astro`: pane header, changelog entries as ANSI date-stamped log lines (e.g., `[2026-03-25] ▶ feat: ...`)
- [x] 4.7 Create `site/src/components/screens/ThemesScreen.astro`: pane header, Dracula palette swatches rendered as ANSI color blocks (`█████`)
- [x] 4.8 Create `site/src/components/HelpOverlay.astro`: full-screen help panel listing all keyboard shortcuts in a box-drawing table

## 5. Pipeline Showcase — ANSI DAG and YAML Typewriter

- [x] 5.1 Write the ANSI DAG diagram content for `PipelinesScreen.astro`: 3-node pipeline (PROVIDER → PLUGIN → OUTPUT) using `╔═╗ ║ ╚═╝ →` characters, colored with `fg-purple`/`fg-cyan`/`fg-green` classes
- [x] 5.2 Write the hardcoded YAML pipeline demo string in `bbs.js` (or a data file): a realistic 20-30 line pipeline with `steps:`, `provider:`, `plugin:`, `input:`, `output:` fields
- [x] 5.3 Adapt `typewriter()` function to support a `<pre>` target element (currently uses `textContent`; ensure newlines render correctly inside `<pre>`)
- [x] 5.4 Wire `PipelinesScreen` `onEnter()` hook to start YAML typewriter; `r` key handler calls reset+replay
- [x] 5.5 Write hacker-voice marketing copy for Pipelines screen: `>` prompt-prefixed paragraphs covering YAML-first design, typed schemas, CLI composability, agent orchestration

## 6. ABBS Rebrand — Copy and Branding Updates

- [x] 6.1 Update `Base.astro` `<title>` template: `ORCAI ABBS — Agentic Bulletin Board System` (remove old "ABS" references)
- [x] 6.2 Update `HomeScreen.astro` hero subtitle from `// ABS NODE 001 //` to `// ABBS NODE 001 //`
- [x] 6.3 Update `AboutScreen.astro` copy: replace any "Agentic Bulletin System" with "Agentic Bulletin Board System"; update feature descriptions to match current TUI (widgets scroll, not pages)
- [x] 6.4 Update `Nav.astro` brand label from `ORCAI ABS` to `ORCAI ABBS`
- [x] 6.5 Review and update all remaining references to "ABS" (not "ABBS") across `site/src/` with a grep pass

## 7. TUI Welcome Dashboard — ABBS Branding

- [x] 7.1 Update `buildWelcomeArt` (or equivalent) in the Go TUI welcome screen to render "ABBS" / "Agentic Bulletin Board System" in the banner subtitle
- [x] 7.2 Update dashboard footer hints to include `q quit · enter connect` alongside existing `^spc n new · ^spc p build` hints

## 8. Cleanup and Migration

- [x] 8.1 Delete `site/src/pages/plugins.astro`, `pipelines.astro`, `getting-started.astro`, `changelog.astro`, `themes.astro` after content migrated to screen components
- [x] 8.2 Update `astro.config.mjs` if needed (single output, base path check)
- [x] 8.3 Run `npm run build` in `site/` and verify no broken imports or build errors
- [x] 8.4 Smoke test: load site in browser, cycle through all 7 screens with number keys, verify no page reload, verify pane scroll isolation, verify YAML typewriter on Pipelines screen
- [x] 8.5 Run the `bbs-site-reviewer` agent to validate Dracula palette compliance and ANSI aesthetic consistency
