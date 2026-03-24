package gitui

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

const refreshInterval = 2 * time.Second

// ─── Domain types ─────────────────────────────────────────────────────────────

type viewMode int

const (
	viewStatus   viewMode = iota
	viewBranches viewMode = iota
	viewCommit   viewMode = iota
	viewFile     viewMode = iota
)

type fileEntry struct {
	path           string
	stagedStatus   string
	worktreeStatus string
}

type gitData struct {
	branch         string
	ahead          int
	behind         int
	lastCommit     string
	unstaged       []fileEntry
	staged         []fileEntry
	localBranches  []string
	remoteBranches []string
}

// ─── Messages ─────────────────────────────────────────────────────────────────

type dataMsg struct{ data *gitData }
type diffMsg struct{ diff string }
type fileContentMsg struct {
	path  string
	lines []string
}
type gitErrMsg struct{ err string }
type statusClearMsg struct{}
type tickMsg time.Time

// ─── Model ────────────────────────────────────────────────────────────────────

type model struct {
	cwd          string
	data         *gitData
	view         viewMode
	cursor       int // index into allFiles()
	branchCursor int // index into combined local+remote branch list
	diff         string
	diffScroll   int // line offset in the diff pane
	focusPane    int // 0 = file list (left), 1 = diff (right)
	commitMsg    string
	errMsg       string
	statusMsg    string
	width        int
	height       int
	loading      bool
	// file viewer
	viewingFile string
	fileLines   []string
	fileScroll  int
}

func New(cwd string) model {
	abs, err := filepath.Abs(cwd)
	if err != nil {
		abs = cwd
	}
	return model{cwd: abs, loading: true}
}

// ─── Init ─────────────────────────────────────────────────────────────────────

func (m model) Init() tea.Cmd {
	return tea.Batch(loadDataCmd(m.cwd), tickCmd())
}

// ─── Update ───────────────────────────────────────────────────────────────────

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case tickMsg:
		// Don't refresh git data while the file viewer is open — avoid
		// background churn competing with the file render goroutine.
		if m.view == viewFile {
			return m, tickCmd()
		}
		return m, tea.Batch(loadDataCmd(m.cwd), tickCmd())

	case dataMsg:
		m.loading = false
		if msg.data != nil {
			m.data = msg.data
			m.errMsg = ""
		} else {
			return m, loadDataCmd(m.cwd)
		}
		return m, diffForCursor(m)

	case diffMsg:
		m.diff = msg.diff
		// Do NOT reset diffScroll here — the tick reloads the diff every 2s
		// and resetting here would jump back to the top on every refresh.
		// diffScroll is only reset when the cursor moves to a different file.

	case fileContentMsg:
		m.fileLines = msg.lines

	case gitErrMsg:
		m.errMsg = msg.err
		m.loading = false

	case statusClearMsg:
		m.statusMsg = ""

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// ── File viewer ───────────────────────────────────────────────────────────
	if m.view == viewFile {
		contentH := m.height - 3
		maxScroll := len(m.fileLines) - contentH
		if maxScroll < 0 {
			maxScroll = 0
		}
		switch msg.String() {
		case "esc", "q":
			m.view = viewStatus
			m.viewingFile = ""
			m.fileLines = nil
			m.fileScroll = 0
		case "j", "down":
			if m.fileScroll < maxScroll {
				m.fileScroll++
			}
		case "k", "up":
			if m.fileScroll > 0 {
				m.fileScroll--
			}
		case "ctrl+d":
			m.fileScroll += contentH / 2
			if m.fileScroll > maxScroll {
				m.fileScroll = maxScroll
			}
		case "ctrl+u":
			m.fileScroll -= contentH / 2
			if m.fileScroll < 0 {
				m.fileScroll = 0
			}
		}
		return m, nil
	}

	// ── Commit view ──────────────────────────────────────────────────────────
	if m.view == viewCommit {
		switch msg.String() {
		case "esc":
			m.view = viewStatus
			m.commitMsg = ""
		case "enter":
			if strings.TrimSpace(m.commitMsg) == "" {
				return m, nil
			}
			text := m.commitMsg
			m.commitMsg = ""
			m.view = viewStatus
			return m, tea.Batch(doCommit(m.cwd, text), clearStatusCmd())
		case "backspace":
			if len(m.commitMsg) > 0 {
				r := []rune(m.commitMsg)
				m.commitMsg = string(r[:len(r)-1])
			}
		default:
			if msg.Type == tea.KeyRunes {
				m.commitMsg += string(msg.Runes)
			}
		}
		return m, nil
	}

	// ── Branch view ──────────────────────────────────────────────────────────
	if m.view == viewBranches {
		total := m.totalBranches()
		switch msg.String() {
		case "esc", "q":
			m.view = viewStatus
		case "j", "down":
			if m.branchCursor < total-1 {
				m.branchCursor++
			}
		case "k", "up":
			if m.branchCursor > 0 {
				m.branchCursor--
			}
		case "enter":
			if branch, ok := m.branchAt(m.branchCursor); ok {
				m.view = viewStatus
				return m, tea.Batch(doCheckout(m.cwd, branch), clearStatusCmd())
			}
		case "f":
			m.statusMsg = "Fetching…"
			return m, tea.Batch(doFetch(m.cwd), clearStatusCmd())
		}
		return m, nil
	}

	// ── Status view ───────────────────────────────────────────────────────────
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "tab":
		// Switch focus between file list (left) and diff (right).
		m.focusPane = 1 - m.focusPane

	case "b":
		m.view = viewBranches
	case "c":
		m.view = viewCommit
	case "r":
		return m, loadDataCmd(m.cwd)
	case "v":
		if m.data != nil {
			all := m.allFiles()
			if m.cursor < len(all) {
				entry := all[m.cursor]
				// Don't try to view directories (git shows them with trailing /).
				fullPath := filepath.Join(m.cwd, entry.path)
				if fi, err := os.Stat(fullPath); err == nil && fi.IsDir() {
					break
				}
				m.view = viewFile
				m.viewingFile = entry.path
				m.fileLines = nil
				m.fileScroll = 0
				m.errMsg = ""
				return m, loadFileCmd(m.cwd, entry.path, m.width)
			}
		}

	case "j", "down":
		if m.focusPane == 1 {
			// Scroll diff down.
			diffLines := strings.Split(m.diff, "\n")
			maxScroll := len(diffLines) - m.diffPageSize()
			if maxScroll < 0 {
				maxScroll = 0
			}
			if m.diffScroll < maxScroll {
				m.diffScroll++
			}
		} else {
			// Move file cursor down.
			if m.cursor < m.totalFiles()-1 {
				m.cursor++
				m.diffScroll = 0
				return m, diffForCursor(m)
			}
		}

	case "k", "up":
		if m.focusPane == 1 {
			// Scroll diff up.
			if m.diffScroll > 0 {
				m.diffScroll--
			}
		} else {
			// Move file cursor up.
			if m.cursor > 0 {
				m.cursor--
				m.diffScroll = 0
				return m, diffForCursor(m)
			}
		}

	case "ctrl+d":
		// Page down in diff (always, regardless of focus).
		diffLines := strings.Split(m.diff, "\n")
		maxScroll := len(diffLines) - m.diffPageSize()
		if maxScroll < 0 {
			maxScroll = 0
		}
		m.diffScroll += m.diffPageSize() / 2
		if m.diffScroll > maxScroll {
			m.diffScroll = maxScroll
		}

	case "ctrl+u":
		// Page up in diff (always, regardless of focus).
		m.diffScroll -= m.diffPageSize() / 2
		if m.diffScroll < 0 {
			m.diffScroll = 0
		}

	case " ":
		if m.data == nil || m.focusPane == 1 {
			return m, nil
		}
		all := m.allFiles()
		if m.cursor >= len(all) {
			return m, nil
		}
		entry := all[m.cursor]
		isStaged := m.cursor >= len(m.data.unstaged)
		return m, doStage(m.cwd, entry, isStaged)
	}
	return m, nil
}

// diffPageSize returns the number of visible lines in the diff pane.
func (m model) diffPageSize() int {
	rows := m.height
	if rows < 20 {
		rows = 20
	}
	size := rows - 6 // header + border top/bottom + footer + filename line
	if size < 4 {
		size = 4
	}
	return size
}

// ─── Branch helpers ───────────────────────────────────────────────────────────

// totalBranches returns the combined count of local + remote branches.
func (m model) totalBranches() int {
	if m.data == nil {
		return 0
	}
	return len(m.data.localBranches) + len(m.data.remoteBranches)
}

// branchAt returns the branch name at position idx in the combined list.
func (m model) branchAt(idx int) (string, bool) {
	if m.data == nil {
		return "", false
	}
	if idx < len(m.data.localBranches) {
		return m.data.localBranches[idx], true
	}
	remote := idx - len(m.data.localBranches)
	if remote < len(m.data.remoteBranches) {
		return m.data.remoteBranches[remote], true
	}
	return "", false
}

// ─── File helpers ─────────────────────────────────────────────────────────────

func (m model) allFiles() []fileEntry {
	if m.data == nil {
		return nil
	}
	return append(append([]fileEntry{}, m.data.unstaged...), m.data.staged...)
}

func (m model) totalFiles() int {
	if m.data == nil {
		return 0
	}
	return len(m.data.unstaged) + len(m.data.staged)
}

// ─── Commands ─────────────────────────────────────────────────────────────────

func loadDataCmd(cwd string) tea.Cmd {
	return func() tea.Msg {
		data, err := loadGitData(cwd)
		if err != nil {
			return gitErrMsg{err.Error()}
		}
		return dataMsg{data}
	}
}

func diffForCursor(m model) tea.Cmd {
	all := m.allFiles()
	if len(all) == 0 || m.cursor >= len(all) {
		return nil
	}
	entry := all[m.cursor]
	isStaged := m.cursor >= len(m.data.unstaged)
	return loadDiff(m.cwd, entry, isStaged)
}

func loadDiff(cwd string, entry fileEntry, isStaged bool) tea.Cmd {
	return func() tea.Msg {
		return diffMsg{getDiff(cwd, entry, isStaged)}
	}
}

func doCommit(cwd, msg string) tea.Cmd {
	return func() tea.Msg {
		if _, err := runGit(cwd, "commit", "-m", msg); err != nil {
			return gitErrMsg{err.Error()}
		}
		return dataMsg{nil}
	}
}

func doStage(cwd string, entry fileEntry, isStaged bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		if isStaged {
			_, err = runGit(cwd, "restore", "--staged", "--", entry.path)
		} else {
			_, err = runGit(cwd, "add", "--", entry.path)
		}
		if err != nil {
			return gitErrMsg{err.Error()}
		}
		return dataMsg{nil}
	}
}

// doCheckout handles both local and remote branches.
// For remote branches (origin/foo), it strips the remote prefix so git can
// auto-create a local tracking branch if one doesn't exist.
func doCheckout(cwd, branch string) tea.Cmd {
	return func() tea.Msg {
		b := branch
		// Strip any remote prefix (origin/foo → foo) so `git checkout foo`
		// will create a local tracking branch automatically.
		if idx := strings.Index(b, "/"); idx >= 0 {
			b = b[idx+1:]
		}
		if _, err := runGit(cwd, "checkout", b); err != nil {
			return gitErrMsg{err.Error()}
		}
		return dataMsg{nil}
	}
}

func doFetch(cwd string) tea.Cmd {
	return func() tea.Msg {
		if _, err := runGit(cwd, "fetch", "--all"); err != nil {
			return gitErrMsg{err.Error()}
		}
		return dataMsg{nil}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func clearStatusCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return statusClearMsg{}
	})
}

// ─── Git helpers ──────────────────────────────────────────────────────────────

func runGit(cwd string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", cwd}, args...)...)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func loadGitData(cwd string) (*gitData, error) {
	d := &gitData{}

	if branch, err := runGit(cwd, "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		d.branch = branch
	} else {
		d.branch = "HEAD"
	}

	if rev, err := runGit(cwd, "rev-list", "--count", "--left-right", "@{u}...HEAD"); err == nil {
		parts := strings.Split(rev, "\t")
		if len(parts) == 2 {
			d.behind, _ = strconv.Atoi(parts[0])
			d.ahead, _ = strconv.Atoi(parts[1])
		}
	}

	d.lastCommit, _ = runGit(cwd, "log", "-1", "--pretty=%s")

	statusOut, err := runGit(cwd, "status", "--short", "--porcelain")
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(statusOut, "\n") {
		if len(line) < 3 {
			continue
		}
		x, y, path := string(line[0]), string(line[1]), line[3:]
		entry := fileEntry{path: path, stagedStatus: x, worktreeStatus: y}
		if x != " " && x != "?" {
			d.staged = append(d.staged, entry)
		}
		if y != " " {
			d.unstaged = append(d.unstaged, entry)
		}
	}

	// Load local branches.
	if out, err := runGit(cwd, "branch", "--format=%(refname:short)"); err == nil {
		for _, b := range strings.Split(out, "\n") {
			if b = strings.TrimSpace(b); b != "" {
				d.localBranches = append(d.localBranches, b)
			}
		}
	}

	// Load remote branches (strip the "remotes/" prefix git sometimes adds).
	if out, err := runGit(cwd, "branch", "-r", "--format=%(refname:short)"); err == nil {
		for _, b := range strings.Split(out, "\n") {
			b = strings.TrimSpace(b)
			b = strings.TrimPrefix(b, "remotes/")
			// Skip HEAD pointers (origin/HEAD -> origin/main).
			if b == "" || strings.HasSuffix(b, "/HEAD") {
				continue
			}
			d.remoteBranches = append(d.remoteBranches, b)
		}
	}

	return d, nil
}

func getDiff(cwd string, entry fileEntry, isStaged bool) string {
	if isStaged {
		out, _ := runGit(cwd, "diff", "--cached", "--", entry.path)
		return out
	}
	if entry.worktreeStatus == "?" {
		content, err := os.ReadFile(filepath.Join(cwd, entry.path))
		if err != nil {
			return ""
		}
		lines := strings.Split(string(content), "\n")
		if len(lines) > 200 {
			lines = lines[:200]
		}
		for i, l := range lines {
			lines[i] = "+" + l
		}
		return strings.Join(lines, "\n")
	}
	out, _ := runGit(cwd, "diff", "--", entry.path)
	return out
}

// ─── Styles ───────────────────────────────────────────────────────────────────

var (
	sNormal    = lipgloss.NewStyle()
	sBold      = lipgloss.NewStyle().Bold(true)
	sMuted     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	sMutedBold = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Bold(true)
	sGreen     = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	sRed       = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	sYellow    = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	sCyan      = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	sCyanBold  = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)

	sBorderInactive = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))
	sBorderActive = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("6")) // cyan when focused
)

func fileStatusStyle(s string) lipgloss.Style {
	switch s {
	case "M":
		return sYellow
	case "A":
		return sGreen
	case "D":
		return sRed
	case "?", "R":
		return sCyan
	default:
		return sNormal
	}
}

func colorDiffLine(line string) string {
	switch {
	case strings.HasPrefix(line, "@@"):
		return sCyan.Render(line)
	case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
		return sGreen.Render(line)
	case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
		return sRed.Render(line)
	case strings.HasPrefix(line, "diff "),
		strings.HasPrefix(line, "index "),
		strings.HasPrefix(line, "---"),
		strings.HasPrefix(line, "+++"):
		return sBold.Render(line)
	default:
		return sMuted.Render(line)
	}
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) > max {
		return "…" + string(r[len(r)-(max-1):])
	}
	return s
}

// truncateLine truncates a diff line from the right, appending "…" if needed.
// Used to prevent long lines from wrapping inside the diff pane.
func truncateLine(s string, max int) string {
	r := []rune(s)
	if len(r) > max {
		return string(r[:max-1]) + "…"
	}
	return s
}

// ─── View ─────────────────────────────────────────────────────────────────────

func (m model) View() string {
	if m.loading && m.data == nil {
		return sMuted.Render("\n  Loading…")
	}
	switch m.view {
	case viewCommit:
		return m.renderCommit()
	case viewBranches:
		return m.renderBranches()
	case viewFile:
		return m.renderFile()
	default:
		return m.renderStatus()
	}
}

func (m model) renderCommit() string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(sBold.Render("Commit") + "\n\n")
	sb.WriteString(sMuted.Render("Staged: "))
	if m.data == nil || len(m.data.staged) == 0 {
		sb.WriteString(sYellow.Render("(none — will use -a)"))
	} else {
		for _, f := range m.data.staged {
			sb.WriteString(sGreen.Render(" " + f.path))
		}
	}
	sb.WriteString("\n\n")

	inputBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("12")).
		Padding(0, 1).
		Width(68)
	sb.WriteString(inputBox.Render(sCyan.Render("Message: ") + m.commitMsg + sBold.Render("|")))
	sb.WriteString("\n\n")
	sb.WriteString(sMuted.Render("[Enter] commit  [Esc] cancel"))
	if m.errMsg != "" {
		sb.WriteString("\n" + sRed.Render(m.errMsg))
	}
	return sb.String()
}

func (m model) renderBranches() string {
	local := []string{}
	remote := []string{}
	if m.data != nil {
		local = m.data.localBranches
		remote = m.data.remoteBranches
	}
	total := len(local) + len(remote)

	rows := m.height - 6
	if rows < 6 {
		rows = 6
	}

	// Calculate visible window centered on cursor.
	start := m.branchCursor - rows/2
	if start < 0 {
		start = 0
	}
	if start > total-rows {
		start = total - rows
	}
	if start < 0 {
		start = 0
	}
	end := start + rows
	if end > total {
		end = total
	}

	currentBranch := ""
	if m.data != nil {
		currentBranch = m.data.branch
	}

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(sBold.Render("Branches") + "  " +
		sMuted.Render(fmt.Sprintf("%d local  %d remote", len(local), len(remote))) + "\n\n")

	// Track whether we've printed the remote header yet.
	remoteDividerPrinted := false

	for i := start; i < end; i++ {
		// Print divider before first remote branch.
		if i == len(local) && !remoteDividerPrinted {
			remoteDividerPrinted = true
			sb.WriteString(sMuted.Render("── remote ──────────────────────") + "\n")
		}

		var name string
		if i < len(local) {
			name = local[i]
		} else {
			name = remote[i-len(local)]
		}

		isCurrent := name == currentBranch ||
			name == "origin/"+currentBranch
		isSelected := i == m.branchCursor

		cur := "  "
		if isSelected {
			cur = "> "
		}
		marker := "  "
		if isCurrent {
			marker = "* "
		}

		displayName := name
		if isCurrent {
			displayName = sGreen.Render(name)
		}

		if isSelected {
			sb.WriteString(sCyanBold.Render(cur+marker) + displayName + "\n")
		} else {
			sb.WriteString(sMuted.Render(cur+marker) + displayName + "\n")
		}
	}

	// Print divider even if it wasn't in the visible window.
	if !remoteDividerPrinted && start >= len(local) {
		// All visible items are remote — print divider at top of list.
	}

	sb.WriteString("\n")
	sb.WriteString(sMuted.Render("[j/k] navigate  [Enter] checkout  [f] fetch  [Esc] back"))
	if m.errMsg != "" {
		sb.WriteString("\n" + sRed.Render(m.errMsg))
	}
	return sb.String()
}

func (m model) renderStatus() string {
	if m.data == nil {
		return sMuted.Render("\n  Loading…")
	}
	d := m.data

	cols := m.width
	if cols < 80 {
		cols = 80
	}
	rows := m.height
	if rows < 20 {
		rows = 20
	}

	leftW := cols * 25 / 100
	rightW := cols - leftW - 4

	// ── Header ────────────────────────────────────────────────────────────────
	var hdr strings.Builder
	hdr.WriteString(sCyanBold.Render(" "+d.branch+"  "))
	if d.ahead > 0 {
		hdr.WriteString(sGreen.Render(fmt.Sprintf("+%d ", d.ahead)))
	}
	if d.behind > 0 {
		hdr.WriteString(sRed.Render(fmt.Sprintf("-%d ", d.behind)))
	}
	if d.ahead == 0 && d.behind == 0 {
		hdr.WriteString(sMuted.Render("up to date  "))
	}
	commit := d.lastCommit
	if len([]rune(commit)) > 60 {
		commit = string([]rune(commit)[:60])
	}
	hdr.WriteString(sMuted.Render(commit))

	// ── Left pane: file list ──────────────────────────────────────────────────
	paneH := rows - 4

	// Build all file lines into a slice so we can window-scroll.
	type fileLine struct{ text string }
	var fileLines []fileLine

	fileLines = append(fileLines, fileLine{sMutedBold.Render(fmt.Sprintf("Unstaged (%d)", len(d.unstaged)))})
	if len(d.unstaged) == 0 {
		fileLines = append(fileLines, fileLine{sMuted.Render("  (clean)")})
	}
	for i, f := range d.unstaged {
		sel := m.cursor == i
		cur := " "
		if sel {
			cur = ">"
		}
		line := fmt.Sprintf("%s %s  %s", cur,
			fileStatusStyle(f.worktreeStatus).Render(f.worktreeStatus),
			truncate(f.path, leftW-7))
		if sel {
			fileLines = append(fileLines, fileLine{sCyanBold.Render(line)})
		} else {
			fileLines = append(fileLines, fileLine{line})
		}
	}
	fileLines = append(fileLines, fileLine{""})
	fileLines = append(fileLines, fileLine{sMutedBold.Render(fmt.Sprintf("Staged (%d)", len(d.staged)))})
	if len(d.staged) == 0 {
		fileLines = append(fileLines, fileLine{sMuted.Render("  (none)")})
	}
	for i, f := range d.staged {
		idx := len(d.unstaged) + i
		sel := m.cursor == idx
		cur := " "
		if sel {
			cur = ">"
		}
		line := fmt.Sprintf("%s %s  %s", cur,
			fileStatusStyle(f.stagedStatus).Render(f.stagedStatus),
			truncate(f.path, leftW-7))
		if sel {
			fileLines = append(fileLines, fileLine{sCyanBold.Render(line)})
		} else {
			fileLines = append(fileLines, fileLine{line})
		}
	}

	// Auto-scroll the file list to keep the selected item visible.
	// Find which line the cursor is on.
	cursorLine := 1 // "Unstaged (n)" header
	if len(d.unstaged) == 0 {
		cursorLine++ // "(clean)"
	}
	if m.cursor < len(d.unstaged) {
		cursorLine += m.cursor
	} else {
		cursorLine += len(d.unstaged) // past all unstaged
		if len(d.unstaged) == 0 {
			cursorLine-- // didn't add (clean) twice
		}
		cursorLine += 2 // blank + "Staged (n)" header
		if len(d.staged) == 0 {
			cursorLine++ // "(none)"
		}
		cursorLine += m.cursor - len(d.unstaged)
	}
	fileScrollOffset := cursorLine - paneH/2
	if fileScrollOffset < 0 {
		fileScrollOffset = 0
	}
	if fileScrollOffset > len(fileLines)-paneH {
		fileScrollOffset = len(fileLines) - paneH
	}
	if fileScrollOffset < 0 {
		fileScrollOffset = 0
	}

	var left strings.Builder
	for i := fileScrollOffset; i < fileScrollOffset+paneH && i < len(fileLines); i++ {
		left.WriteString(fileLines[i].text + "\n")
	}

	leftBorder := sBorderInactive
	if m.focusPane == 0 {
		leftBorder = sBorderActive
	}
	leftPane := leftBorder.Width(leftW - 2).Height(paneH).Render(left.String())

	// ── Right pane: diff ──────────────────────────────────────────────────────
	all := m.allFiles()
	diffH := paneH - 1 // subtract 1 for the filename header line

	// maxLineW is the usable inner width of the right pane.
	// rightW - 2 (border) - 1 (small margin) to prevent lipgloss wrapping.
	maxLineW := rightW - 3
	if maxLineW < 20 {
		maxLineW = 20
	}

	var right strings.Builder
	if m.cursor < len(all) {
		right.WriteString(sMutedBold.Render(truncate(all[m.cursor].path, maxLineW)) + "\n")
		lines := strings.Split(m.diff, "\n")
		scrollable := len(lines) > diffH
		// When scrollable, reserve the last line for the percentage indicator
		// so total content stays within diffH (paneH - 1 for the filename).
		contentLines := diffH
		if scrollable {
			contentLines = diffH - 1
		}
		for i := m.diffScroll; i < m.diffScroll+contentLines && i < len(lines); i++ {
			right.WriteString(colorDiffLine(truncateLine(lines[i], maxLineW)) + "\n")
		}
		if scrollable {
			pct := (m.diffScroll + contentLines) * 100 / len(lines)
			if pct > 100 {
				pct = 100
			}
			right.WriteString(sMuted.Render(fmt.Sprintf("── %d%% ──", pct)))
		}
	} else {
		right.WriteString(sMuted.Render("Select a file to see diff"))
	}

	rightBorder := sBorderInactive
	if m.focusPane == 1 {
		rightBorder = sBorderActive
	}
	rightPane := rightBorder.Width(rightW - 2).Height(paneH).Render(right.String())

	// ── Footer ────────────────────────────────────────────────────────────────
	var footer string
	switch {
	case m.statusMsg != "":
		footer = sGreen.Render(m.statusMsg)
	case m.errMsg != "":
		footer = sRed.Render(m.errMsg)
	case m.focusPane == 1:
		footer = sMuted.Render("[j/k] scroll diff  [ctrl+d/u] page  [Tab] focus files  [q] quit")
	default:
		footer = sMuted.Render("[j/k] navigate  [space] stage/unstage  [v] view  [c] commit  [b] branches  [Tab] focus diff  [r] refresh  [q] quit")
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	return lipgloss.JoinVertical(lipgloss.Left, hdr.String(), body, footer)
}

// ─── File viewer ──────────────────────────────────────────────────────────────

const maxFileBytes = 512 * 1024 // 512 KB — cap to keep rendering fast

func loadFileCmd(cwd, path string, termWidth int) tea.Cmd {
	return func() (msg tea.Msg) {
		// Catch any panic from glamour/chroma so we always return something.
		defer func() {
			if r := recover(); r != nil {
				msg = fileContentMsg{path: path, lines: []string{fmt.Sprintf("render error: %v", r)}}
			}
		}()

		fullPath := filepath.Join(cwd, path)
		f, err := os.Open(fullPath)
		var content string
		var truncated bool
		if err != nil {
			// File may have been deleted — try to read from git HEAD.
			out, gitErr := runGit(cwd, "show", "HEAD:"+path)
			if gitErr != nil {
				return fileContentMsg{path: path, lines: []string{"(deleted — not in HEAD either)"}}
			}
			if len(out) > maxFileBytes {
				out = out[:maxFileBytes]
				truncated = true
			}
			content = out
		} else {
			defer f.Close()
			raw, readErr := io.ReadAll(io.LimitReader(f, maxFileBytes+1))
			if readErr != nil {
				return fileContentMsg{path: path, lines: []string{"error: " + readErr.Error()}}
			}
			truncated = len(raw) > maxFileBytes
			if truncated {
				raw = raw[:maxFileBytes]
			}
			content = string(raw)
		}

		width := termWidth - 4
		if width < 40 {
			width = 40
		}

		var rendered string
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".md", ".markdown", ".mdx":
			rendered = renderMarkdown(content, width)
		default:
			rendered = highlightCode(path, content)
		}

		if truncated {
			rendered += "\n\n[truncated at 512 KB]"
		}

		lines := strings.Split(strings.ReplaceAll(rendered, "\r\n", "\n"), "\n")
		return fileContentMsg{path: path, lines: lines}
	}
}

func renderMarkdown(content string, width int) string {
	// Use an explicit dark style — WithAutoStyle() probes the terminal and
	// can hang or panic inside the alt-screen context.
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return content
	}
	out, err := r.Render(content)
	if err != nil {
		return content
	}
	return out
}

func highlightCode(filename, content string) string {
	lexer := lexers.Match(filename)
	if lexer == nil {
		lexer = lexers.Analyse(content)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get("dracula")
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("terminal16m")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		return content
	}

	var buf strings.Builder
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return content
	}
	return buf.String()
}

func (m model) renderFile() string {
	cols := m.width
	if cols < 80 {
		cols = 80
	}
	rows := m.height
	if rows < 20 {
		rows = 20
	}

	contentH := rows - 3 // header row + footer row + 1 buffer

	var sb strings.Builder

	// Header
	sb.WriteString(sCyanBold.Render(" "+m.viewingFile) + "  " +
		sMuted.Render("[j/k] scroll  [ctrl+d/u] page  [Esc/q] back") + "\n")

	if len(m.fileLines) == 0 {
		sb.WriteString(sMuted.Render("\n  Loading…") + "\n")
		if m.errMsg != "" {
			sb.WriteString(sRed.Render("  " + m.errMsg))
		}
		return sb.String()
	}

	// Content — cap line width to prevent wrapping
	maxW := cols - 2
	for i := m.fileScroll; i < m.fileScroll+contentH-1 && i < len(m.fileLines); i++ {
		line := m.fileLines[i]
		// Only truncate plain lines; ANSI-coded lines from chroma/glamour are
		// already wrapped to width, so we skip truncation for them.
		if !strings.Contains(line, "\x1b") {
			line = truncateLine(line, maxW)
		}
		sb.WriteString(line + "\n")
	}

	// Footer: scroll percentage
	if len(m.fileLines) > contentH {
		pct := (m.fileScroll + contentH) * 100 / len(m.fileLines)
		if pct > 100 {
			pct = 100
		}
		sb.WriteString(sMuted.Render(fmt.Sprintf("── %d%% ── %d lines", pct, len(m.fileLines))))
	}

	return sb.String()
}

// ─── Run ──────────────────────────────────────────────────────────────────────

func Run(cwd string) error {
	p := tea.NewProgram(New(cwd), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
