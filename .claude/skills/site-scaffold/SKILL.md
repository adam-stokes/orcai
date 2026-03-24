---
name: site-scaffold
description: Scaffold or regenerate the orcai GitHub Pages site structure at docs/ with full BBS/ANSI aesthetic — Dracula palette, hex-dump canvas wallpaper, monospace grid, ANSI art hero, VT323 headers, dot-separated nav. Use when adding a new page or resetting the site structure.
disable-model-invocation: true
---

Scaffold or update the orcai GitHub Pages site at docs/.

When invoked with a page name as argument, create that page. With no argument, scaffold the full site.

## Site Structure

docs/
  index.html              — BBS welcome screen, ANSI logo hero, hex dump bg
  getting-started.html    — Terminal-style install + first run guide
  plugins.html            — Plugin system with ASCII architecture diagrams
  pipelines.html          — Pipeline builder YAML reference
  css/bbs.css             — Dracula palette, CRT scanlines, BBS components
  js/bbs.js               — Hex canvas, typewriter, copy buttons, keyboard nav
  ans/                    — Converted ANSI art HTML fragments
  _config.yml             — GitHub Pages config (theme: null)
  .nojekyll               — Disable Jekyll

## Aesthetic Rules (ENFORCE ON ALL PAGES)

- Background: #282a36 ONLY. No white, light, or gradient backgrounds.
- Fonts: VT323 (Google Fonts) for display/headers. Share Tech Mono for body.
- Nav: Fixed top bar, 1px border in --purple, dot-separated links
- Box-drawing: Use ║╔╗╚╝─│┌┐└┘· for all UI frames
- Colors: Only Dracula palette vars (--purple, --pink, --cyan, --green, --yellow, --red, --comment)
- NO Bootstrap, NO Tailwind, NO utility frameworks
- CRT scanlines via CSS ::after on body
- Vignette via CSS ::before on body

## When Adding a New Page

1. Read docs/index.html to understand the nav structure
2. Read docs/css/bbs.css for existing styles to reuse
3. Create the new page following the same header/nav/footer pattern
4. Add the page to the nav bar in ALL existing pages
5. Follow the BBS content style: ASCII boxes, terminal prompts, monospace tables

## Palette Reference (from sdk/styles/styles.go)

--bg: #282a36 | --fg: #f8f8f2 | --purple: #bd93f9 | --pink: #ff79c6
--cyan: #8be9fd | --green: #50fa7b | --yellow: #f1fa8c | --red: #ff5555
--comment: #6272a4 | --selbg: #44475a | --darkbg: #1e1f29
