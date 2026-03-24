package promptbuilder

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/adam-stokes/orcai/internal/pipeline"
)

var (
	borderStyle  = lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(lipgloss.Color("63"))
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	selectedStep = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	dimStep      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	statusBar    = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
	labelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
)

// BubbleModel wraps Model and implements tea.Model.
type BubbleModel struct {
	inner  *Model
	width  int
	height int
}

// NewBubble creates a bubbletea-compatible model.
func NewBubble(m *Model) *BubbleModel {
	return &BubbleModel{inner: m}
}

func (b *BubbleModel) Init() tea.Cmd { return nil }

func (b *BubbleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		b.width = msg.Width
		b.height = msg.Height
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return b, tea.Quit
		case key.Matches(msg, keys.Up):
			b.inner.SelectStep(b.inner.SelectedIndex() - 1)
		case key.Matches(msg, keys.Down):
			b.inner.SelectStep(b.inner.SelectedIndex() + 1)
		case key.Matches(msg, keys.AddStep):
			id := fmt.Sprintf("step%d", len(b.inner.Steps())+1)
			b.inner.AddStep(pipeline.Step{ID: id, Plugin: "claude"})
		}
	}
	return b, nil
}

func (b *BubbleModel) View() string {
	if b.width == 0 {
		return "Loading..."
	}

	w := b.width * 80 / 100
	h := b.height * 80 / 100
	leftW := w * 30 / 100
	rightW := w - leftW - 4

	// Left pane: step list.
	leftContent := titleStyle.Render("STEPS") + "\n" + strings.Repeat("─", leftW-2) + "\n"
	for i, s := range b.inner.Steps() {
		label := fmt.Sprintf("[%d] %s", i+1, stepLabel(s))
		if i == b.inner.SelectedIndex() {
			leftContent += selectedStep.Render("→ "+label) + "\n"
		} else {
			leftContent += dimStep.Render("  "+label) + "\n"
		}
	}
	leftContent += "\n" + dimStep.Render("[+] add step")

	// Right pane: config for selected step.
	rightContent := ""
	steps := b.inner.Steps()
	if len(steps) > 0 {
		sel := steps[b.inner.SelectedIndex()]
		rightContent = titleStyle.Render(fmt.Sprintf("STEP %d — CONFIG", b.inner.SelectedIndex()+1)) + "\n"
		rightContent += strings.Repeat("─", rightW-2) + "\n"
		rightContent += labelStyle.Render("ID:      ") + sel.ID + "\n"
		rightContent += labelStyle.Render("Plugin:  ") + sel.Plugin + "\n"
		rightContent += labelStyle.Render("Model:   ") + sel.Model + "\n"
		rightContent += labelStyle.Render("Prompt:  ") + sel.Prompt + "\n"
		if sel.Condition.If != "" {
			rightContent += labelStyle.Render("Cond:    ") + sel.Condition.If + "\n"
			rightContent += labelStyle.Render("  then→  ") + sel.Condition.Then + "\n"
			rightContent += labelStyle.Render("  else→  ") + sel.Condition.Else + "\n"
		}
	}

	left := lipgloss.NewStyle().Width(leftW).Height(h - 6).Render(leftContent)
	right := lipgloss.NewStyle().Width(rightW).Height(h - 6).Render(rightContent)
	panes := lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)

	header := titleStyle.Render("PIPELINE BUILDER") +
		lipgloss.NewStyle().Width(w-20).Render("") +
		dimStep.Render("[?] help  [x]")
	nameRow := labelStyle.Render("NAME: ") + b.inner.Name()
	footer := statusBar.Render("[r] run  [s] save  [tab] next field  [↑↓] steps  [esc] quit")

	modal := borderStyle.Width(w).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			header,
			nameRow,
			strings.Repeat("═", w-4),
			panes,
			strings.Repeat("═", w-4),
			footer,
		),
	)

	marginLeft := (b.width - w) / 2
	marginTop := (b.height - h) / 2
	return lipgloss.NewStyle().
		MarginLeft(marginLeft).
		MarginTop(marginTop).
		Render(modal)
}

func stepLabel(s pipeline.Step) string {
	if s.Type != "" {
		return s.Type
	}
	if s.Plugin != "" {
		return s.Plugin
	}
	return s.ID
}
