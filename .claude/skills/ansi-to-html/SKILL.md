---
name: ansi-to-html
description: Convert an ANSI art .ans file to a web-embeddable HTML fragment with Dracula CSS color classes. Also generates the Go cmd/ansi2html/main.go converter for use in the Makefile and GitHub Actions.
---

Convert ANSI art files to web-embeddable HTML for the orcai GitHub Pages site.

When invoked with a file path as argument, convert that file.
With no argument, convert all files in assets/ui/*.ans.

## Conversion Rules

Parse ANSI CSI escape sequences and emit HTML:

1. SGR color codes → CSS classes:
   - \x1b[0m → reset (close all spans)
   - \x1b[1m → class="bold"
   - \x1b[5m → class="blink"
   - \x1b[3Xm (foreground, X=0-7) → class="fg-{color-name}"
   - \x1b[9Xm (bright foreground) → class="fg-bright-{color-name}"
   - \x1b[4Xm (background) → class="bg-{color-name}"
   - \x1b[38;5;Nm (256-color) → style="color:{dracula-mapped-color}"

2. Dracula color mapping for standard 16 colors:
   0=bg(#282a36), 1=red(#ff5555), 2=green(#50fa7b), 3=yellow(#f1fa8c),
   4=purple(#bd93f9), 5=pink(#ff79c6), 6=cyan(#8be9fd), 7=fg(#f8f8f2)
   8=comment(#6272a4), 9-15=bright variants

3. Wrap result in: <pre class="ansi-art"><code>...</code></pre>

4. Output: docs/ans/<name>.html

## Also Generate cmd/ansi2html/main.go

A Go CLI tool that performs the same conversion:
- Usage: ./bin/ansi2html <input.ans> <output.html>
- Reads .ans file, parses ANSI codes, emits HTML fragment
- Used by: Makefile target `make ansi2html` and GitHub Actions

Add Makefile target:
```
ansi2html: ## Convert ANSI art assets to HTML fragments
	go build -o bin/ansi2html ./cmd/ansi2html/
	for f in assets/ui/*.ans; do \
		name=$$(basename $$f .ans); \
		./bin/ansi2html $$f docs/ans/$$name.html; \
	done
```
