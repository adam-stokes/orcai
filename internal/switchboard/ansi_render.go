package switchboard

import (
	"github.com/adam-stokes/orcai/internal/panelrender"
	"github.com/adam-stokes/orcai/internal/themes"
)

// visibleWidth returns the printable character count of s, stripping ANSI escapes.
// Used by internal tests; delegates to panelrender.
func visibleWidth(s string) int { return panelrender.VisibleWidth(s) }

// RenderHeader returns the translated (or plain-text fallback) title for a panel.
// Used in boxTop() and DynamicHeader() when no ANS sprite is available.
// If a GlobalProvider is set, the panel's translation key is looked up first.
func RenderHeader(panel string) string { return panelrender.RenderHeader(panel) }

// SpriteLines returns the ANS sprite for a panel as individual lines, ready
// to be prepended in place of a boxTop() call.
func SpriteLines(bundle *themes.Bundle, panel string, panelWidth int) []string {
	return panelrender.SpriteLines(bundle, panel, panelWidth)
}

// DynamicHeader generates a 3-line panel header at exactly width visible
// columns, using colors from bundle.HeaderStyle.
func DynamicHeader(bundle *themes.Bundle, panel string, width int, borderColor string) []string {
	return panelrender.DynamicHeader(bundle, panel, width, borderColor)
}

// PanelHeader returns the best available header for a panel at the given width.
// It tries fixed-width .ans sprites first (SpriteLines), then falls back to
// DynamicHeader which always produces the correct panel width.
// Returns nil only when both sources are unavailable.
func PanelHeader(bundle *themes.Bundle, panel string, width int, borderColor string) []string {
	return panelrender.PanelHeader(bundle, panel, width, borderColor)
}
