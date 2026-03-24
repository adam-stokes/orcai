package picker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/adam-stokes/orcai/internal/opsx"
	"github.com/adam-stokes/orcai/sdk/styles"
)

// ModelOption is a selectable model within a provider.
type ModelOption struct {
	ID        string
	Label     string
	Separator bool // visual divider — not selectable
}

// ProviderDef describes one AI provider and its available models.
type ProviderDef struct {
	ID     string
	Label  string
	Models []ModelOption
}

// Providers is the canonical ordered list. Models for ollama are discovered at
// runtime; this list is used as the base before availability filtering.
var Providers = []ProviderDef{
	{
		ID: "claude", Label: "Claude",
		Models: []ModelOption{
			{ID: "claude-opus-4-6", Label: "Opus 4.6"},
			{ID: "claude-sonnet-4-6", Label: "Sonnet 4.6"},
			{ID: "claude-haiku-4-5-20251001", Label: "Haiku 4.5"},
		},
	},
	{ID: "opencode", Label: "OpenCode"},
	{ID: "copilot", Label: "GitHub Copilot"},
	{ID: "ollama", Label: "Ollama"},
	{ID: "shell", Label: "Shell"},
}

// isInstalled reports whether cmd exists in PATH.
func isInstalled(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// queryOllamaModels calls the local Ollama API and returns model names.
// Returns nil if Ollama is not running or has no models.
func queryOllamaModels() []string {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := client.Get("http://localhost:11434/api/tags")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}
	names := make([]string, 0, len(result.Models))
	for _, m := range result.Models {
		names = append(names, m.Name)
	}
	return names
}

// ensureContextWindows ensures each Ollama model has a -ctx32k variant with
// num_ctx=32768. Returns the list of context-extended model names, falling back
// to the original name if creation fails. Creation is instant (metadata only).
func ensureContextWindows(models []string) []string {
	const numCtx = 32768
	const suffix = "-ctx32k"

	existing := make(map[string]bool, len(models))
	for _, m := range models {
		existing[m] = true
	}

	client := &http.Client{Timeout: 10 * time.Second}
	result := make([]string, 0, len(models))

	for _, m := range models {
		// Split name and tag: "qwen3.5:latest" → name="qwen3.5", tag="latest"
		name, tag := m, "latest"
		if idx := strings.LastIndex(m, ":"); idx >= 0 {
			name, tag = m[:idx], m[idx+1:]
		}
		ctxModel := name + suffix + ":" + tag

		if !existing[ctxModel] {
			modelfile := fmt.Sprintf("FROM %s\nPARAMETER num_ctx %d", m, numCtx)
			body, _ := json.Marshal(map[string]string{
				"model":     ctxModel,
				"modelfile": modelfile,
			})
			// Drain the streaming response; creation is near-instant for metadata changes.
			if resp, err := client.Post("http://localhost:11434/api/create",
				"application/json", bytes.NewReader(body)); err == nil {
				resp.Body.Close()
				existing[ctxModel] = true
			}
		}

		if existing[ctxModel] {
			result = append(result, ctxModel)
		} else {
			result = append(result, m) // creation failed — use original
		}
	}
	return result
}

// buildProviders returns a filtered, runtime-enriched provider list:
//   - only includes providers whose CLI is installed (shell always included)
//   - injects discovered Ollama models into the ollama and opencode providers
func buildProviders() []ProviderDef {
	ollamaModels := queryOllamaModels()

	// For opencode, create ctx32k variants so agentic tasks have enough context.
	var opencodeModels []string
	if len(ollamaModels) > 0 && isInstalled("opencode") {
		opencodeModels = ensureContextWindows(ollamaModels)
		ensureOpencodeOllamaConfig(opencodeModels)
	}

	var out []ProviderDef
	for _, p := range Providers {
		if p.ID != "shell" && !isInstalled(p.ID) {
			continue
		}
		p = injectOllamaModels(p, ollamaModels, opencodeModels)
		out = append(out, p)
	}
	return out
}

// ensureOpencodeOllamaConfig writes (or merges) the ollama provider block into
// ~/.config/opencode/opencode.json so opencode can reach local Ollama models.
// The model ID format opencode expects is "ollama/<model-name>".
func ensureOpencodeOllamaConfig(ollamaModels []string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	cfgPath := filepath.Join(home, ".config", "opencode", "opencode.json")

	// Read existing config or start fresh.
	var cfg map[string]interface{}
	if data, err := os.ReadFile(cfgPath); err == nil {
		json.Unmarshal(data, &cfg) //nolint:errcheck
	}
	if cfg == nil {
		cfg = map[string]interface{}{}
	}

	// Build the models map: { "qwen3.5:latest": { "name": "qwen3.5:latest" } }
	models := make(map[string]interface{}, len(ollamaModels))
	for _, m := range ollamaModels {
		models[m] = map[string]interface{}{"name": m}
	}

	// Merge provider block — preserve any other providers already configured.
	providers, _ := cfg["provider"].(map[string]interface{})
	if providers == nil {
		providers = map[string]interface{}{}
	}
	providers["ollama"] = map[string]interface{}{
		"npm":  "@ai-sdk/openai-compatible",
		"name": "Ollama (local)",
		"options": map[string]interface{}{
			"baseURL": "http://localhost:11434/v1",
		},
		"models": models,
	}
	cfg["$schema"] = "https://opencode.ai/config.json"
	cfg["provider"] = providers

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return
	}
	os.MkdirAll(filepath.Dir(cfgPath), 0o755)    //nolint:errcheck
	os.WriteFile(cfgPath, data, 0o644)            //nolint:errcheck
}

// injectOllamaModels appends Ollama model entries to providers that support them.
// ollamaModels are the raw model names (used for standalone ollama).
// opencodeModels are the ctx32k variants (used for opencode — may be nil).
func injectOllamaModels(p ProviderDef, ollamaModels, opencodeModels []string) ProviderDef {
	switch p.ID {
	case "ollama":
		if len(ollamaModels) == 0 {
			return p
		}
		models := make([]ModelOption, 0, len(ollamaModels))
		for _, m := range ollamaModels {
			models = append(models, ModelOption{ID: m, Label: m})
		}
		p.Models = models

	case "opencode":
		if len(opencodeModels) == 0 {
			return p
		}
		models := make([]ModelOption, len(p.Models))
		copy(models, p.Models)
		if len(models) > 0 {
			models = append(models, ModelOption{Separator: true, Label: "── Ollama ──"})
		}
		for _, m := range opencodeModels {
			// Strip the -ctx32k suffix for the display label.
			label := strings.Replace(m, "-ctx32k", "", 1)
			models = append(models, ModelOption{ID: "ollama/" + m, Label: label})
		}
		p.Models = models
	}
	return p
}

// ── Worktree helpers ────────────────────────────────────────────────────────

// currentPanePath returns the filesystem path of the active tmux pane.
func currentPanePath() string {
	out, err := exec.Command("tmux", "display-message", "-p", "#{pane_current_path}").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// findGitRoot returns the top-level git directory containing path, or "".
func findGitRoot(path string) string {
	if path == "" {
		return ""
	}
	out, err := exec.Command("git", "-C", path, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// GetOrCreateWorktreeFrom creates a git worktree for sessionName rooted at
// basePath and returns its path plus the repo root. Returns ("", "") if
// basePath is empty or not inside a git repo, or ("", repoRoot) if worktree
// creation fails — callers fall back to the repo root directory.
func GetOrCreateWorktreeFrom(basePath, sessionName string) (worktreePath, repoRoot string) {
	repoRoot = findGitRoot(basePath)
	if repoRoot == "" {
		return "", ""
	}

	// Place the worktree as a sibling of the repo: <parent>/<repoName>-<session>
	worktreePath = filepath.Join(
		filepath.Dir(repoRoot),
		filepath.Base(repoRoot)+"-"+sessionName,
	)

	// Reuse an existing worktree path rather than erroring.
	if _, err := os.Stat(worktreePath); err == nil {
		return worktreePath, repoRoot
	}

	// Try to create with a named branch so sessions are traceable.
	branch := "orcai/" + sessionName
	if err := exec.Command("git", "-C", repoRoot, "worktree", "add", worktreePath, "-b", branch).Run(); err != nil {
		// Branch already exists or some other issue — fall back to detached HEAD.
		if err2 := exec.Command("git", "-C", repoRoot, "worktree", "add", "--detach", worktreePath).Run(); err2 != nil {
			return "", repoRoot // worktree creation failed; caller uses repoRoot
		}
	}
	return worktreePath, repoRoot
}

// copyDotEnv copies .env from src directory to dst directory if the file
// exists in src and does not already exist in dst.
func copyDotEnv(src, dst string) {
	srcFile := filepath.Join(src, ".env")
	dstFile := filepath.Join(dst, ".env")
	if _, err := os.Stat(srcFile); err != nil {
		return // no .env to copy
	}
	if _, err := os.Stat(dstFile); err == nil {
		return // dst already has a .env
	}
	data, err := os.ReadFile(srcFile)
	if err != nil {
		return
	}
	os.WriteFile(dstFile, data, 0o600) //nolint:errcheck
}

// ── Google Cloud env helpers ─────────────────────────────────────────────────

// gcpEnvArgs returns tmux -e KEY=VALUE args for Google Cloud env vars that are
// set in the current process environment. Used to forward gcloud credentials
// and project config into gemini sessions running in worktrees.
var gcpEnvKeys = []string{
	"GOOGLE_CLOUD_PROJECT",
	"GOOGLE_CLOUD_LOCATION",
	"GOOGLE_APPLICATION_CREDENTIALS",
	"CLOUDSDK_CORE_PROJECT",
	"CLOUDSDK_COMPUTE_REGION",
	"GCLOUD_PROJECT",
}

func gcpEnvArgs() []string {
	var args []string
	for _, k := range gcpEnvKeys {
		if v := os.Getenv(k); v != "" {
			args = append(args, "-e", k+"="+v)
		}
	}
	return args
}

// ── Existing session helpers ─────────────────────────────────────────────────

// WindowEntry represents a running orcai tmux window shown in the session picker.
type WindowEntry struct {
	Index string
	Name  string
}

// systemWindows are orcai UI windows that should not appear in the existing sessions list.
var systemWindows = map[string]bool{
	"ORCAI":    true,
	"_sidebar": true,
	"_welcome": true,
}

// ParseWindowList parses the output of:
//
//	tmux list-windows -t orcai -F "#{window_index} #{window_name}"
//
// and returns non-system windows.
func ParseWindowList(output string) []WindowEntry {
	var entries []WindowEntry
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx, name, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}
		if systemWindows[name] {
			continue
		}
		entries = append(entries, WindowEntry{Index: idx, Name: name})
	}
	return entries
}

// listExistingSessions queries tmux for running orcai windows.
func listExistingSessions() []WindowEntry {
	out, err := exec.Command("tmux", "list-windows", "-t", "orcai",
		"-F", "#{window_index} #{window_name}").Output()
	if err != nil {
		return nil
	}
	return ParseWindowList(string(out))
}

// focusWindow switches the orcai session to the window with the given index.
func focusWindow(index string) {
	exec.Command("tmux", "select-window", "-t", "orcai:"+index).Run() //nolint:errcheck
}

// ── Picker TUI ───────────────────────────────────────────────────────────────

// PickerState represents which screen the picker is currently showing.
type PickerState int

const (
	StateSearch       PickerState = iota // fuzzy list (initial state)
	StateProvider                        // pick which CLI to run skill/agent with
	StateModel                           // pick model for a provider
	StateWorkdir                         // working directory input
	StateWorkflow                        // fresh vs openspec choice
	StateOpenSpecName                    // openspec feature name input
)

type pickerModel struct {
	providers        []ProviderDef
	sessions         []WindowEntry  // populated at init
	selectedSession  *WindowEntry   // non-nil when user picked an existing session
	state            PickerState
	pCursor          int
	mCursor          int
	width            int
	height           int
	quit             bool
	workdirInput     textinput.Model
	selectedProvider ProviderDef
	selectedModelID  string
	openspecInput     textinput.Model // text input for feature name
	openspecFeature   string          // confirmed feature slug (set before launch)
	openspecAvailable bool            // true when openspec CLI is in PATH
	wfCursor          int             // cursor for workflow choice (0=fresh, 1=openspec)
	launchedWorktree  string          // worktree path created by doLaunch (may be "")
}

func newPickerModel() pickerModel {
	ti := textinput.New()
	ti.Placeholder = "/path/to/project"
	ti.CharLimit = 256

	oi := textinput.New()
	oi.Placeholder = "e.g. add-login-flow"
	oi.CharLimit = 80

	_, openspecErr := exec.LookPath("openspec")
	return pickerModel{
		providers:         buildProviders(),
		sessions:          listExistingSessions(),
		workdirInput:      ti,
		openspecInput:     oi,
		openspecAvailable: openspecErr == nil,
	}
}

func (m pickerModel) Init() tea.Cmd { return textinput.Blink }

// selectableModels returns only non-separator entries.
func selectableModels(p ProviderDef) []ModelOption {
	var out []ModelOption
	for _, mo := range p.Models {
		if !mo.Separator {
			out = append(out, mo)
		}
	}
	return out
}

// resolveProviderCursor returns the selected WindowEntry (if in the sessions
// section) or the selected ProviderDef. Exactly one of the return values is valid.
func (m pickerModel) resolveProviderCursor() (session *WindowEntry, provider *ProviderDef) {
	if m.pCursor < len(m.sessions) {
		return &m.sessions[m.pCursor], nil
	}
	p := m.providers[m.pCursor-len(m.sessions)]
	return nil, &p
}

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		// Workdir state handles keys independently.
		if m.state == StateWorkdir {
			switch msg.String() {
			case "ctrl+c":
				m.quit = true
				return m, tea.Quit
			case "esc":
				if len(m.selectedProvider.Models) > 0 {
					m.state = StateModel
				} else {
					m.state = StateProvider
				}
				m.workdirInput.Blur()
				return m, nil
			case "enter":
				if !m.openspecAvailable {
					m.doLaunch()
					return m, tea.Quit
				}
				m.wfCursor = 0
				m.openspecFeature = ""
				m.state = StateWorkflow
				return m, nil
			default:
				var cmd tea.Cmd
				m.workdirInput, cmd = m.workdirInput.Update(msg)
				return m, cmd
			}
		}

		if m.state == StateWorkflow {
			switch msg.String() {
			case "ctrl+c":
				m.quit = true
				return m, tea.Quit
			case "j", "down":
				maxCursor := 0
				if m.openspecAvailable {
					maxCursor = 1
				}
				if m.wfCursor < maxCursor {
					m.wfCursor++
				}
			case "k", "up":
				if m.wfCursor > 0 {
					m.wfCursor--
				}
			case "enter":
				if m.wfCursor == 1 {
					m.openspecInput.SetValue("")
					m.openspecInput.Focus()
					m.state = StateOpenSpecName
				} else {
					m.doLaunch()
					return m, tea.Quit
				}
			case "esc":
				if m.selectedSession != nil {
					m.selectedSession = nil
					m.state = StateProvider
				} else {
					m.workdirInput.Focus()
					m.state = StateWorkdir
				}
			}
			return m, nil
		}

		if m.state == StateOpenSpecName {
			switch msg.String() {
			case "ctrl+c":
				m.quit = true
				return m, tea.Quit
			case "enter":
				slug := opsx.FeatureSlug(m.openspecInput.Value())
				if slug != "" {
					m.openspecFeature = slug
					m.doLaunch()
					return m, tea.Quit
				}
			case "esc":
				m.openspecInput.Blur()
				m.state = StateWorkflow
			default:
				var cmd tea.Cmd
				m.openspecInput, cmd = m.openspecInput.Update(msg)
				return m, cmd
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			m.quit = true
			return m, tea.Quit

		case "esc":
			if m.state == StateModel {
				m.state = StateProvider
				m.mCursor = 0
			} else {
				m.quit = true
				return m, tea.Quit
			}

		case "j", "down":
			if m.state == StateProvider {
				if m.pCursor < len(m.sessions)+len(m.providers)-1 {
					m.pCursor++
				}
			} else {
				sel := selectableModels(m.providers[m.pCursor-len(m.sessions)])
				if m.mCursor < len(sel)-1 {
					m.mCursor++
				}
			}

		case "k", "up":
			if m.state == StateProvider {
				if m.pCursor > 0 {
					m.pCursor--
				}
			} else {
				if m.mCursor > 0 {
					m.mCursor--
				}
			}

		case "enter":
			if m.state == StateProvider {
				session, provider := m.resolveProviderCursor()
				if session != nil {
					// Existing session selected: focus the window and go to workflow.
					m.selectedSession = session
					focusWindow(session.Index)
					m.workdirInput.SetValue(currentPanePath())
					if !m.openspecAvailable {
						return m, tea.Quit
					}
					m.wfCursor = 0
					m.openspecFeature = ""
					m.state = StateWorkflow
					return m, nil
				} else if len(provider.Models) > 0 {
					m.state = StateModel
					m.mCursor = 0
				} else {
					// Provider with no models: go straight to workdir.
					m.selectedProvider = *provider
					m.selectedModelID = ""
					m.workdirInput.SetValue(currentPanePath())
					m.workdirInput.Focus()
					m.state = StateWorkdir
				}
			} else {
				// StateModel
				p := m.providers[m.pCursor-len(m.sessions)]
				modelID := selectableModels(p)[m.mCursor].ID
				m.selectedProvider = p
				m.selectedModelID = modelID
				m.workdirInput.SetValue(currentPanePath())
				m.workdirInput.Focus()
				m.state = StateWorkdir
			}
		}
	}
	return m, nil
}

func (m pickerModel) View() string {
	if m.quit {
		return ""
	}

	w := m.width
	if w <= 0 {
		w = 42
	}

	headerStyle := lipgloss.NewStyle().
		Background(styles.Purple).
		Foreground(styles.Bg).
		Bold(true).
		Width(w).
		Padding(0, 1)

	activeStyle := lipgloss.NewStyle().
		Background(styles.SelBg).
		Foreground(styles.Pink).
		Width(w).
		Padding(0, 1)

	inactiveStyle := lipgloss.NewStyle().
		Foreground(styles.Fg).
		Width(w).
		Padding(0, 1)

	sepStyle := lipgloss.NewStyle().
		Foreground(styles.Comment).
		Width(w).
		Padding(0, 1)

	footerStyle := lipgloss.NewStyle().
		Foreground(styles.Comment).
		Width(w).
		Padding(0, 1)

	var rows []string

	sectionStyle := lipgloss.NewStyle().
		Foreground(styles.Comment).
		Width(w).
		Padding(0, 1)

	switch m.state {
	case StateProvider:
		rows = append(rows, headerStyle.Render("ORCAI  New Session"))

		if len(m.sessions) > 0 {
			rows = append(rows, sectionStyle.Render("── existing ──"))
			for i, s := range m.sessions {
				if m.pCursor == i {
					rows = append(rows, activeStyle.Render("▎ "+s.Name))
				} else {
					rows = append(rows, inactiveStyle.Render("  "+s.Name))
				}
			}
			rows = append(rows, sectionStyle.Render("── new ──"))
		}

		for i, p := range m.providers {
			label := p.Label
			if len(p.Models) > 0 {
				label += " ›"
			}
			idx := len(m.sessions) + i
			if idx == m.pCursor {
				rows = append(rows, activeStyle.Render("▎ "+label))
			} else {
				rows = append(rows, inactiveStyle.Render("  "+label))
			}
		}
		rows = append(rows, footerStyle.Render("↑↓ nav  enter select  q quit"))

	case StateModel:
		p := m.providers[m.pCursor-len(m.sessions)]
		rows = append(rows, headerStyle.Render("ORCAI  "+p.Label+" › Model"))
		cursor := 0
		for _, mo := range p.Models {
			if mo.Separator {
				rows = append(rows, sepStyle.Render("  "+mo.Label))
				continue
			}
			if cursor == m.mCursor {
				rows = append(rows, activeStyle.Render("▎ "+mo.Label))
			} else {
				rows = append(rows, inactiveStyle.Render("  "+mo.Label))
			}
			cursor++
		}
		rows = append(rows, footerStyle.Render("↑↓ nav  enter select  esc back"))

	case StateWorkdir:
		rows = append(rows, headerStyle.Render("ORCAI  Working Directory"))

		bodyStyle := lipgloss.NewStyle().Width(w).Padding(1, 2)
		labelStyle := lipgloss.NewStyle().Foreground(styles.Fg).Bold(true)
		body := lipgloss.JoinVertical(lipgloss.Left,
			labelStyle.Render("Base directory:"),
			m.workdirInput.View(),
		)
		rows = append(rows, bodyStyle.Render(body))
		rows = append(rows, footerStyle.Render("enter confirm  esc back"))

	case StateWorkflow:
		rows = append(rows, headerStyle.Render("ORCAI  Workflow"))
		choices := []string{"Start fresh"}
		if m.openspecAvailable {
			choices = append(choices, "Start with OpenSpec")
		}
		for i, c := range choices {
			if i == m.wfCursor {
				rows = append(rows, activeStyle.Render("▎ "+c))
			} else {
				rows = append(rows, inactiveStyle.Render("  "+c))
			}
		}
		rows = append(rows, footerStyle.Render("↑↓ nav  enter select  esc back"))

	case StateOpenSpecName:
		rows = append(rows, headerStyle.Render("ORCAI  OpenSpec"))
		bodyStyle := lipgloss.NewStyle().Width(w).Padding(1, 2)
		labelStyle := lipgloss.NewStyle().Foreground(styles.Fg).Bold(true)
		body := lipgloss.JoinVertical(lipgloss.Left,
			labelStyle.Render("Feature name:"),
			m.openspecInput.View(),
		)
		rows = append(rows, bodyStyle.Render(body))
		rows = append(rows, footerStyle.Render("enter confirm  esc back"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// doLaunch performs the session launch from pickerModel state and stores the
// worktree path so Run() can pass it to opsx.ProviderSend.
func (m *pickerModel) doLaunch() {
	if m.selectedSession != nil {
		return // existing session already focused; Run() handles ProviderSend
	}
	basePath := strings.TrimSpace(m.workdirInput.Value())
	if basePath == "" {
		basePath = currentPanePath()
	}
	m.launchedWorktree = launchFrom(m.selectedProvider, m.selectedModelID, basePath)
}

// ── Session launch ───────────────────────────────────────────────────────────

// launchFrom creates a tmux window for the chosen provider + model, rooted in a
// fresh git worktree derived from basePath when it is inside a git repository.
// Returns the worktree path (or repo root if no worktree was created, or ""
// if basePath is not inside a git repository).
func launchFrom(p ProviderDef, modelID, basePath string) string {
	// Unique window name.
	out, _ := exec.Command("tmux", "list-windows", "-t", "orcai", "-F", "#{window_name}").Output()
	count := 0
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.HasPrefix(line, p.ID+"-") || line == p.ID {
			count++
		}
	}
	name := fmt.Sprintf("%s-%d", p.ID, count+1)

	// Create (or reuse) a git worktree for this session.
	worktreePath, repoRoot := GetOrCreateWorktreeFrom(basePath, name)
	startDir := worktreePath
	if startDir == "" {
		startDir = repoRoot // not in git repo, or worktree creation failed
	}

	// Copy .env from repo root into the fresh worktree so provider configs
	// (GOOGLE_CLOUD_PROJECT, etc.) are available without re-creating them.
	if worktreePath != "" && repoRoot != "" {
		copyDotEnv(repoRoot, worktreePath)
	}

	// Build tmux new-window args.
	args := []string{"new-window", "-t", "orcai", "-n", name}
	if startDir != "" {
		args = append(args, "-c", startDir)
	}

	// Forward Google Cloud credentials + project config for gemini sessions.
	if p.ID == "gemini" {
		args = append(args, gcpEnvArgs()...)
	}

	// Build the shell command.
	switch p.ID {
	case "shell":
		exec.Command("tmux", args...).Run() //nolint:errcheck
		return startDir
	case "ollama":
		args = append(args, "ollama run "+modelID)
	default:
		shellCmd := p.ID
		if modelID != "" {
			shellCmd = p.ID + " --model " + modelID
		}
		args = append(args, shellCmd)
	}

	exec.Command("tmux", args...).Run() //nolint:errcheck
	return startDir
}

// Run displays the provider/model picker in a bubbletea program.
func Run() {
	p := tea.NewProgram(newPickerModel(), tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		fmt.Printf("picker error: %v\n", err)
		return
	}
	// Run the OpenSpec propose workflow after the popup is fully closed, giving
	// the newly launched CLI process time to initialise before receiving input.
	if pm, ok := result.(pickerModel); ok && pm.openspecFeature != "" {
		opsx.ProviderSend(pm.openspecFeature, pm.selectedProvider.ID, pm.launchedWorktree)
	}
}
