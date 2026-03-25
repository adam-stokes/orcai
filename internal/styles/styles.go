// Package styles provides Dracula-themed lipgloss styles for orcai UIs.
package styles

import "github.com/charmbracelet/lipgloss"

// Dracula colour palette
var (
	Purple  = lipgloss.Color("#bd93f9")
	Pink    = lipgloss.Color("#ff79c6")
	Cyan    = lipgloss.Color("#8be9fd")
	Green   = lipgloss.Color("#50fa7b")
	Yellow  = lipgloss.Color("#f1fa8c")
	Red     = lipgloss.Color("#ff5555")
	Comment = lipgloss.Color("#6272a4")
	Fg      = lipgloss.Color("#f8f8f2")
	Bg      = lipgloss.Color("#282a36")
	SelBg   = lipgloss.Color("#44475a")
)

// Pre-built styles
var (
	Title    = lipgloss.NewStyle().Foreground(Purple).Bold(true)
	Subtitle = lipgloss.NewStyle().Foreground(Cyan)
	Selected = lipgloss.NewStyle().Background(SelBg).Foreground(Pink)
	Dimmed   = lipgloss.NewStyle().Foreground(Comment)
	Normal   = lipgloss.NewStyle().Foreground(Fg)
	Success  = lipgloss.NewStyle().Foreground(Green)
	Error    = lipgloss.NewStyle().Foreground(Red)
	Warning  = lipgloss.NewStyle().Foreground(Yellow)
	Divider  = lipgloss.NewStyle().Foreground(Comment)
	Border   = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Purple)
)
