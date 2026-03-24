// Package uiassets provides default ANSI art constants for ORCAI's TUI.
// User overrides are supported via ~/.config/orcai/ui/ (handled by ansiart.Load).
package uiassets

// welcomeANSI is the full-width welcome banner (~52 visible cols wide, 6 rows).
// Inner box width: 50 visible chars between the ║ borders.
const welcomeANSI = "" +
	"\x1b[38;5;141m╔══════════════════════════════════════════════════╗\x1b[0m\n" +
	"\x1b[38;5;141m║\x1b[38;5;212m ░▒▓ \x1b[1;38;5;212mO R C A I\x1b[0m\x1b[38;5;212m ▓▒░\x1b[38;5;61m  Your AI Workspace             \x1b[38;5;141m║\x1b[0m\n" +
	"\x1b[38;5;141m║\x1b[38;5;61m      tmux · AI agents · open sessions            \x1b[38;5;141m║\x1b[0m\n" +
	"\x1b[38;5;141m╠══════════════════════════════════════════════════╣\x1b[0m\n" +
	"\x1b[38;5;141m║\x1b[38;5;212m▄▀▄▀▄▀▄▀▄▀▄▀▄▀▄▀▄▀▄▀▄▀▄▀▄▀▄▀▄▀▄▀▄▀▄▀▄▀▄▀▄▀▄▀▄▀▄▀▄▀\x1b[38;5;141m║\x1b[0m\n" +
	"\x1b[38;5;141m╚══════════════════════════════════════════════════╝\x1b[0m"

// WelcomeAns is the default welcome ANSI art, ready for use with ansiart.Load.
var WelcomeAns = []byte(welcomeANSI)
