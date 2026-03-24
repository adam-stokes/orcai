package ollama

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type uiState int

const (
	uiLoading uiState = iota
	uiUnavailable
	uiList
	uiBuildConfig
	uiBuilding
	uiDone
	uiErr
)

var ctxSizes = []int{8, 16, 32, 64, 128}

// TUIModel is the Bubble Tea model for the Ollama model manager.
type TUIModel struct {
	state     uiState
	models    []LocalModel
	cursor    int
	ctxCursor int
	selected  LocalModel
	built     string
	err       error
	spin      spinner.Model
	width     int
	height    int
}

// messages
type modelsLoadedMsg struct{ models []LocalModel }
type buildDoneMsg struct{ name string }
type buildErrMsg struct{ err error }

func NewTUI() TUIModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("99"))
	return TUIModel{state: uiLoading, ctxCursor: 1, spin: s}
}

func (m TUIModel) Init() tea.Cmd {
	return tea.Batch(m.spin.Tick, loadModelsCmd())
}

func loadModelsCmd() tea.Cmd {
	return func() tea.Msg {
		if !IsAvailable() {
			return buildErrMsg{fmt.Errorf("ollama is not running (http://localhost:11434)")}
		}
		models, err := ListModels()
		if err != nil {
			return buildErrMsg{err}
		}
		return modelsLoadedMsg{models}
	}
}

func doBuildCmd(baseModel string, ctxK int) tea.Cmd {
	return func() tea.Msg {
		if err := BuildCtxModel(baseModel, ctxK); err != nil {
			return buildErrMsg{err}
		}
		name := CtxVariantName(baseModel, ctxK)
		_ = UpdateOpencodeConfig(name) // best-effort
		return buildDoneMsg{name}
	}
}

func (m TUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case modelsLoadedMsg:
		// Only show base models — ctx variants are not valid build targets.
		for _, lm := range msg.models {
			if !IsCtxVariant(lm.Name) {
				m.models = append(m.models, lm)
			}
		}
		m.state = uiList

	case buildDoneMsg:
		m.built = msg.name
		m.state = uiDone

	case buildErrMsg:
		m.err = msg.err
		m.state = uiErr

	case tea.KeyMsg:
		switch m.state {
		case uiList:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "j", "down":
				if m.cursor < len(m.models)-1 {
					m.cursor++
				}
			case "k", "up":
				if m.cursor > 0 {
					m.cursor--
				}
			case "enter":
				if len(m.models) > 0 {
					m.selected = m.models[m.cursor]
					m.state = uiBuildConfig
				}
			}

		case uiBuildConfig:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.state = uiList
			case "j", "down":
				if m.ctxCursor < len(ctxSizes)-1 {
					m.ctxCursor++
				}
			case "k", "up":
				if m.ctxCursor > 0 {
					m.ctxCursor--
				}
			case "enter":
				m.state = uiBuilding
				return m, tea.Batch(m.spin.Tick, doBuildCmd(m.selected.Name, ctxSizes[m.ctxCursor]))
			}

		case uiDone, uiErr:
			return m, tea.Quit
		}
	}
	return m, nil
}

// Styles
var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	activeStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	mutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	ctxTagStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	successStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	errStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	hintStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
)

func (m TUIModel) View() string {
	var sb strings.Builder

	switch m.state {
	case uiLoading:
		sb.WriteString(titleStyle.Render("Ollama Models") + "\n\n")
		sb.WriteString("  " + m.spin.View() + " connecting...\n")

	case uiList:
		sb.WriteString(titleStyle.Render("Ollama Models") + "\n\n")
		if len(m.models) == 0 {
			sb.WriteString(mutedStyle.Render("  No local models found.") + "\n")
		} else {
			for i, lm := range m.models {
				cursor := "  "
				name := lm.Name
				if i == m.cursor {
					cursor = "> "
					name = activeStyle.Render(name)
				}
				tag := ""
				if IsCtxVariant(lm.Name) {
					tag = " " + ctxTagStyle.Render("[ctx]")
				}
				size := mutedStyle.Render(fmt.Sprintf("  %.1f GB", float64(lm.Size)/1e9))
				sb.WriteString(fmt.Sprintf("  %s%s%s%s\n", cursor, name, tag, size))
			}
		}
		sb.WriteString("\n" + hintStyle.Render("  enter:build ctx  j/k:move  q:quit"))

	case uiBuildConfig:
		variantPreview := CtxVariantName(m.selected.Name, ctxSizes[m.ctxCursor])
		sb.WriteString(titleStyle.Render("Build extended context") + "\n")
		sb.WriteString(mutedStyle.Render("  model: ") + m.selected.Name + "\n\n")
		for i, size := range ctxSizes {
			cursor := "  "
			label := fmt.Sprintf("%dk  (%d tokens)", size, size*1024)
			if i == m.ctxCursor {
				cursor = "> "
				label = activeStyle.Render(label)
			}
			sb.WriteString(fmt.Sprintf("  %s%s\n", cursor, label))
		}
		sb.WriteString("\n  " + mutedStyle.Render("→ ") + variantPreview + "\n")
		sb.WriteString("\n" + hintStyle.Render("  enter:build  esc:back  q:quit"))

	case uiBuilding:
		name := CtxVariantName(m.selected.Name, ctxSizes[m.ctxCursor])
		sb.WriteString(titleStyle.Render("Building...") + "\n\n")
		sb.WriteString("  " + m.spin.View() + " " + name + "\n")
		sb.WriteString(mutedStyle.Render("  copying model layers, please wait\n"))

	case uiDone:
		sb.WriteString(successStyle.Render("  ✓ "+m.built+" ready") + "\n")
		sb.WriteString(successStyle.Render("  ✓ opencode config updated") + "\n\n")
		sb.WriteString(hintStyle.Render("  any key to close"))

	case uiErr:
		sb.WriteString(errStyle.Render("  ✗ "+m.err.Error()) + "\n\n")
		sb.WriteString(hintStyle.Render("  any key to close"))
	}

	return sb.String()
}
