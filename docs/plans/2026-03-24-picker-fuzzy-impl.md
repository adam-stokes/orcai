# Picker Fuzzy Search & Unified Session Launcher Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the wizard-style provider picker with a single fuzzy-searchable list covering existing sessions, pipelines, skills, agents, and providers — all grouped and filterable by typing.

**Architecture:** Add `sahilm/fuzzy` for substring matching. A new `items.go` builds a flat `[]PickerItem` from all sources (sessions, discovery pipelines, chatui.ScanIndex skills/agents, buildProviders). The picker's initial state changes from `StateProvider` (linear list) to `StateSearch` (fuzzy input + grouped list). Picking a skill/agent routes to a repurposed `StateProvider` screen (pick which CLI), then workdir, then launches the CLI and injects the skill/agent inject text via `tmux send-keys` after a startup delay.

**Tech Stack:** Go, `github.com/sahilm/fuzzy`, `charmbracelet/bubbletea`, `charmbracelet/lipgloss`, existing `chatui.ScanIndex`, `discovery.Discover`, `buildProviders()`

---

### Task 1: Add sahilm/fuzzy dependency

**Files:**
- Modify: `go.mod`, `go.sum`

**Step 1: Add the dependency**

```bash
cd /Users/stokes/Projects/orcai
go get github.com/sahilm/fuzzy@latest
go mod tidy
```

**Step 2: Verify build is clean**

```bash
go build ./...
```

Expected: no errors.

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add sahilm/fuzzy for picker fuzzy filtering"
```

---

### Task 2: PickerItem type, ApplyFuzzy, and BuildPickerItems

**Files:**
- Create: `internal/picker/items.go`
- Create: `internal/picker/items_test.go`

**Step 1: Write the failing tests**

Create `internal/picker/items_test.go`:

```go
package picker_test

import (
	"testing"

	"github.com/adam-stokes/orcai/internal/picker"
)

func TestApplyFuzzy_EmptyQuery(t *testing.T) {
	items := []picker.PickerItem{
		{Kind: "skill", Name: "golang-patterns", Description: "Idiomatic Go"},
		{Kind: "skill", Name: "golang-testing", Description: "Go testing"},
	}
	got := picker.ApplyFuzzy("", items)
	if len(got) != 2 {
		t.Fatalf("empty query: want 2 items, got %d", len(got))
	}
	for _, item := range got {
		if len(item.MatchIndexes()) != 0 {
			t.Errorf("empty query: expected no match indexes, got %v", item.MatchIndexes())
		}
	}
}

func TestApplyFuzzy_FiltersMatches(t *testing.T) {
	items := []picker.PickerItem{
		{Kind: "skill", Name: "golang-patterns", Description: "Idiomatic Go"},
		{Kind: "agent", Name: "beast-mode", Description: "coding agent"},
		{Kind: "pipeline", Name: "research-pipeline", Description: "research"},
	}
	got := picker.ApplyFuzzy("beast", items)
	if len(got) != 1 {
		t.Fatalf("want 1 match for 'beast', got %d", len(got))
	}
	if got[0].Name != "beast-mode" {
		t.Errorf("want beast-mode, got %q", got[0].Name)
	}
}

func TestApplyFuzzy_MatchIndexesSet(t *testing.T) {
	items := []picker.PickerItem{
		{Kind: "skill", Name: "golang-patterns", Description: ""},
	}
	got := picker.ApplyFuzzy("go", items)
	if len(got) == 0 {
		t.Fatal("expected match for 'go'")
	}
	if len(got[0].MatchIndexes()) == 0 {
		t.Error("expected match indexes to be set after fuzzy match")
	}
}

func TestApplyFuzzy_NoMatch(t *testing.T) {
	items := []picker.PickerItem{
		{Kind: "skill", Name: "golang-patterns", Description: ""},
	}
	got := picker.ApplyFuzzy("zzzzz", items)
	if len(got) != 0 {
		t.Errorf("expected no matches, got %d", len(got))
	}
}

func TestBuildPickerItems_HasProviders(t *testing.T) {
	providers := []picker.ProviderDef{
		{ID: "claude", Label: "Claude", Models: []picker.ModelOption{{ID: "claude-sonnet-4-6", Label: "Sonnet"}}},
		{ID: "shell", Label: "Shell"},
	}
	items := picker.BuildPickerItems(nil, providers, "/tmp", "/tmp")
	var found int
	for _, item := range items {
		if item.Kind == "provider" {
			found++
		}
	}
	if found != 2 {
		t.Errorf("want 2 provider items, got %d", found)
	}
}

func TestBuildPickerItems_SessionsFirst(t *testing.T) {
	sessions := []picker.WindowEntry{{Index: "1", Name: "claude-1"}}
	items := picker.BuildPickerItems(sessions, nil, "/tmp", "/tmp")
	if len(items) == 0 {
		t.Fatal("expected items")
	}
	if items[0].Kind != "session" {
		t.Errorf("first item should be session, got %q", items[0].Kind)
	}
}

func TestBuildPickerItems_ProvidersLast(t *testing.T) {
	providers := []picker.ProviderDef{{ID: "shell", Label: "Shell"}}
	items := picker.BuildPickerItems(nil, providers, "/tmp", "/tmp")
	if len(items) == 0 {
		t.Fatal("expected items")
	}
	last := items[len(items)-1]
	if last.Kind != "provider" {
		t.Errorf("last item group should be provider, got %q", last.Kind)
	}
}

func TestPickerItem_FilterString(t *testing.T) {
	item := picker.PickerItem{Kind: "skill", Name: "beast-mode", Description: "top-notch coding agent"}
	got := item.Filter()
	want := "beast-mode top-notch coding agent"
	if got != want {
		t.Errorf("Filter() = %q, want %q", got, want)
	}
}
```

**Step 2: Run tests — confirm they fail**

```bash
go test ./internal/picker/... -run "TestApplyFuzzy|TestBuildPickerItems|TestPickerItem" -v 2>&1 | head -20
```

Expected: FAIL — `picker.PickerItem`, `picker.ApplyFuzzy`, `picker.BuildPickerItems` undefined.

**Step 3: Write the implementation**

Create `internal/picker/items.go`:

```go
package picker

import (
	"os"

	"github.com/sahilm/fuzzy"

	"github.com/adam-stokes/orcai/internal/chatui"
	"github.com/adam-stokes/orcai/internal/discovery"
)

// PickerItem is a single selectable row in the fuzzy picker.
type PickerItem struct {
	Kind         string // "session" | "pipeline" | "skill" | "agent" | "provider"
	Name         string
	Description  string
	SourceTag    string // "[global]" "[project]" "[copilot]" — empty for providers/sessions
	ProviderID   string // for kind=provider; also set after skill/agent picks a CLI
	ModelID      string // for kind=provider with a pre-selected model
	InjectText   string // for kind=skill|agent — sent to CLI after launch via tmux send-keys
	PipelineFile string // for kind=pipeline
	SessionIndex string // for kind=session — tmux window index to focus
	// internal — populated by ApplyFuzzy
	matchIndexes []int
}

// Filter returns the string used for fuzzy matching.
func (p PickerItem) Filter() string { return p.Name + " " + p.Description }

// SetMatchIndexes stores which character positions were matched by the fuzzy algorithm.
func (p *PickerItem) SetMatchIndexes(indexes []int) { p.matchIndexes = indexes }

// MatchIndexes returns the stored fuzzy match positions (nil when no filter active).
func (p PickerItem) MatchIndexes() []int { return p.matchIndexes }

// itemsSource implements fuzzy.Source over a []PickerItem.
type itemsSource []PickerItem

func (s itemsSource) Len() int            { return len(s) }
func (s itemsSource) String(i int) string { return s[i].Filter() }

// ApplyFuzzy filters items using sahilm/fuzzy.
// Returns all items (group order preserved) when query is empty.
// Returns matched items sorted by score when query is non-empty.
func ApplyFuzzy(query string, items []PickerItem) []PickerItem {
	if query == "" {
		out := make([]PickerItem, len(items))
		for i, item := range items {
			item.matchIndexes = nil
			out[i] = item
		}
		return out
	}
	matches := fuzzy.FindFrom(query, itemsSource(items))
	out := make([]PickerItem, len(matches))
	for i, m := range matches {
		item := items[m.Index]
		item.matchIndexes = m.MatchedIndexes
		out[i] = item
	}
	return out
}

// orcaiConfigDir returns ~/.config/orcai, or "" on error.
func orcaiConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home + "/.config/orcai"
}

// BuildPickerItems assembles all session-starter items in display group order:
// sessions → pipelines → skills → agents → providers.
// cwd and homeDir are passed to chatui.ScanIndex to locate skills and agents.
func BuildPickerItems(sessions []WindowEntry, providers []ProviderDef, cwd, homeDir string) []PickerItem {
	var items []PickerItem

	// ── sessions ──
	for _, s := range sessions {
		items = append(items, PickerItem{
			Kind:         "session",
			Name:         s.Name,
			Description:  "existing session",
			SessionIndex: s.Index,
		})
	}

	// ── pipelines ── (TypePipeline only — native/CLI-wrapper entries overlap providers)
	if configDir := orcaiConfigDir(); configDir != "" {
		if plugins, err := discovery.Discover(configDir); err == nil {
			for _, p := range plugins {
				if p.Type != discovery.TypePipeline {
					continue
				}
				items = append(items, PickerItem{
					Kind:         "pipeline",
					Name:         p.Name,
					Description:  "pipeline",
					PipelineFile: p.PipelineFile,
				})
			}
		}
	}

	// ── skills + agents ──
	index := chatui.ScanIndex(cwd, homeDir)
	for _, e := range index {
		if e.Kind != "skill" && e.Kind != "agent" {
			continue
		}
		items = append(items, PickerItem{
			Kind:        e.Kind,
			Name:        e.Name,
			Description: e.Description,
			SourceTag:   chatui.SourceLabel(e.Source),
			InjectText:  e.Inject,
		})
	}

	// ── providers ──
	for _, p := range providers {
		desc := ""
		if len(selectableModels(p)) > 0 {
			desc = "select model"
		}
		items = append(items, PickerItem{
			Kind:        "provider",
			Name:        p.Label,
			Description: desc,
			ProviderID:  p.ID,
		})
	}

	return items
}
```

**Step 4: Run tests — confirm they pass**

```bash
go test ./internal/picker/... -run "TestApplyFuzzy|TestBuildPickerItems|TestPickerItem" -v
```

Expected: all PASS.

**Step 5: Commit**

```bash
git add internal/picker/items.go internal/picker/items_test.go
git commit -m "feat(picker): add PickerItem, ApplyFuzzy, and BuildPickerItems"
```

---

### Task 3: Add StateSearch constant and update state test

**Files:**
- Modify: `internal/picker/picker.go` (the `PickerState` const block)
- Modify: `internal/picker/picker_test.go` (`TestPickerStates_AllDistinct`)

**Step 1: Write the failing test first**

In `internal/picker/picker_test.go`, update `TestPickerStates_AllDistinct` to include `StateSearch`:

```go
func TestPickerStates_AllDistinct(t *testing.T) {
	states := []picker.PickerState{
		picker.StateSearch,
		picker.StateProvider,
		picker.StateModel,
		picker.StateWorkdir,
		picker.StateWorkflow,
		picker.StateOpenSpecName,
	}
	seen := map[picker.PickerState]bool{}
	for _, s := range states {
		if seen[s] {
			t.Errorf("duplicate picker state value: %v", s)
		}
		seen[s] = true
	}
}
```

**Step 2: Run test — confirm it fails**

```bash
go test ./internal/picker/... -run TestPickerStates_AllDistinct -v
```

Expected: FAIL — `picker.StateSearch` undefined.

**Step 3: Add StateSearch to the const block in picker.go**

Find this block in `internal/picker/picker.go` (around line 393):

```go
const (
	StateProvider    PickerState = iota
	StateModel
	StateWorkdir
	StateWorkflow     // NEW
	StateOpenSpecName // NEW
)
```

Replace with:

```go
const (
	StateSearch       PickerState = iota // fuzzy list (initial state)
	StateProvider                        // pick which CLI to run skill/agent with
	StateModel                           // pick model for a provider
	StateWorkdir                         // working directory input
	StateWorkflow                        // fresh vs openspec choice
	StateOpenSpecName                    // openspec feature name input
)
```

**Step 4: Run test — confirm it passes**

```bash
go test ./internal/picker/... -run TestPickerStates_AllDistinct -v
```

Expected: PASS.

**Step 5: Run full picker test suite to catch regressions**

```bash
go test ./internal/picker/... -v
```

Some tests may now fail because `pickerModel.state` starts at `StateSearch` (0) instead of `StateProvider` (was 0, now 1). That is expected — we fix the model in Task 4.

**Step 6: Commit**

```bash
git add internal/picker/picker.go internal/picker/picker_test.go
git commit -m "feat(picker): add StateSearch as initial picker state"
```

---

### Task 4: Integrate fuzzy search into pickerModel — fields, Init, Update

**Files:**
- Modify: `internal/picker/picker.go`

This task rewires the `pickerModel` struct, `newPickerModel`, and `Update` to use `StateSearch` as the initial state and handle the new skill/agent → provider sub-flow.

**Step 1: Add fields to pickerModel**

Find the `pickerModel` struct (around line 401) and add these fields after `openspecAvailable bool`:

```go
// ── fuzzy search (StateSearch) ──
searchInput   textinput.Model
allItems      []PickerItem  // full item list built at init
filteredItems []PickerItem  // result of ApplyFuzzy on allItems
itemCursor    int           // cursor position in filteredItems

// ── skill/agent provider picker (StateProvider) ──
selectedItem   *PickerItem  // item picked in StateSearch
skillProviders []ProviderDef // installed CLIs shown when launching skill/agent
spCursor       int          // cursor for skill provider picker
```

**Step 2: Update newPickerModel to initialize search and items**

Find `newPickerModel()` and add after the existing `ti` and `oi` setup (before the `return`):

```go
si := textinput.New()
si.Placeholder = "search skills, agents, pipelines, providers..."
si.CharLimit = 80
si.Focus()

cwd, _ := os.Getwd()
home, _ := os.UserHomeDir()
provs := buildProviders()
sessions := listExistingSessions()
all := BuildPickerItems(sessions, provs, cwd, home)
```

Update the `return` statement to include the new fields:

```go
return pickerModel{
    providers:         provs,
    sessions:          sessions,
    workdirInput:      ti,
    openspecInput:     oi,
    openspecAvailable: openspecErr == nil,
    searchInput:       si,
    allItems:          all,
    filteredItems:     ApplyFuzzy("", all),
    skillProviders:    provs,
}
```

Note: the `state` field defaults to zero-value `StateSearch` — no need to set it explicitly.

**Step 3: Add StateSearch handler at the top of Update**

In `Update`, before the existing `if m.state == StateWorkdir {` block, insert:

```go
if m.state == StateSearch {
    switch msg.String() {
    case "ctrl+c", "q":
        m.quit = true
        return m, tea.Quit

    case "j", "down":
        if m.itemCursor < len(m.filteredItems)-1 {
            m.itemCursor++
        }

    case "k", "up":
        if m.itemCursor > 0 {
            m.itemCursor--
        }

    case "enter":
        if len(m.filteredItems) == 0 {
            return m, nil
        }
        item := m.filteredItems[m.itemCursor]
        m.selectedItem = &item
        switch item.Kind {
        case "session":
            focusWindow(item.SessionIndex)
            m.quit = true
            return m, tea.Quit

        case "pipeline":
            m.workdirInput.SetValue(currentPanePath())
            m.workdirInput.Focus()
            m.state = StateWorkdir

        case "skill", "agent":
            m.spCursor = 0
            m.state = StateProvider

        case "provider":
            for _, p := range m.providers {
                if p.ID == item.ProviderID {
                    m.selectedProvider = p
                    break
                }
            }
            if len(selectableModels(m.selectedProvider)) > 0 {
                m.mCursor = 0
                m.state = StateModel
            } else {
                m.selectedModelID = ""
                m.workdirInput.SetValue(currentPanePath())
                m.workdirInput.Focus()
                m.state = StateWorkdir
            }
        }

    default:
        var cmd tea.Cmd
        m.searchInput, cmd = m.searchInput.Update(msg)
        m.filteredItems = ApplyFuzzy(m.searchInput.Value(), m.allItems)
        m.itemCursor = 0
        return m, cmd
    }
    return m, nil
}
```

**Step 4: Add StateProvider handler for skill/agent CLI picker**

After the `StateSearch` block and before `StateWorkdir`, insert:

```go
// StateProvider is repurposed: pick which installed CLI to use for a skill/agent launch.
// Only entered when m.selectedItem.Kind is "skill" or "agent".
if m.state == StateProvider {
    switch msg.String() {
    case "ctrl+c":
        m.quit = true
        return m, tea.Quit

    case "esc":
        m.selectedItem = nil
        m.state = StateSearch
        m.searchInput.Focus()

    case "j", "down":
        if m.spCursor < len(m.skillProviders)-1 {
            m.spCursor++
        }

    case "k", "up":
        if m.spCursor > 0 {
            m.spCursor--
        }

    case "enter":
        if len(m.skillProviders) > 0 {
            m.selectedProvider = m.skillProviders[m.spCursor]
            m.selectedModelID = ""
            m.workdirInput.SetValue(currentPanePath())
            m.workdirInput.Focus()
            m.state = StateWorkdir
        }
    }
    return m, nil
}
```

**Step 5: Update StateWorkdir to skip OpenSpec workflow for skill/agent/pipeline launches**

In the existing `StateWorkdir` → `"enter"` case, the current code transitions to `StateWorkflow` when openspec is available. Skills and pipelines should bypass this:

Find:
```go
case "enter":
    if !m.openspecAvailable {
        m.doLaunch()
        return m, tea.Quit
    }
    m.wfCursor = 0
    m.openspecFeature = ""
    m.state = StateWorkflow
    return m, nil
```

Replace with:
```go
case "enter":
    // Skills, agents, and pipelines bypass the OpenSpec workflow.
    if m.selectedItem != nil {
        m.doLaunch()
        return m, tea.Quit
    }
    if !m.openspecAvailable {
        m.doLaunch()
        return m, tea.Quit
    }
    m.wfCursor = 0
    m.openspecFeature = ""
    m.state = StateWorkflow
    return m, nil
```

**Step 6: Update StateWorkflow esc to return to StateSearch**

In the `StateWorkflow` → `"esc"` case, the existing code returns to `StateProvider` or `StateWorkdir`. Update:

Find:
```go
case "esc":
    if m.selectedSession != nil {
        m.selectedSession = nil
        m.state = StateProvider
    } else {
        m.workdirInput.Focus()
        m.state = StateWorkdir
    }
```

Replace with:
```go
case "esc":
    if m.selectedSession != nil {
        m.selectedSession = nil
        m.state = StateSearch
        m.searchInput.Focus()
    } else {
        m.workdirInput.Focus()
        m.state = StateWorkdir
    }
```

**Step 7: Run all picker tests**

```bash
go test ./internal/picker/... -v
```

Expected: all PASS (or only view-related tests failing, which we fix in Task 5).

**Step 8: Commit**

```bash
git add internal/picker/picker.go
git commit -m "feat(picker): wire StateSearch and StateProvider into pickerModel Update"
```

---

### Task 5: Update View() — StateSearch rendering and repurposed StateProvider view

**Files:**
- Modify: `internal/picker/picker.go` (`View()` method)

**Step 1: Add renderMatchHighlights helper (before View)**

Add this function anywhere in `picker.go` before `View()`:

```go
// renderMatchHighlights returns name with matched characters highlighted in pink ANSI.
// When matchIndexes is empty, the name is returned unchanged.
func renderMatchHighlights(name string, matchIndexes []int) string {
	if len(matchIndexes) == 0 {
		return name
	}
	matched := make(map[int]bool, len(matchIndexes))
	for _, idx := range matchIndexes {
		matched[idx] = true
	}
	const pink = "\x1b[38;5;212m"
	const reset = "\x1b[0m"
	var sb strings.Builder
	for i, r := range name {
		if matched[i] {
			sb.WriteString(pink + string(r) + reset)
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
```

**Step 2: Replace StateProvider case in View() with StateSearch + repurposed StateProvider**

In `View()`, find the `switch m.state {` block. Replace the `case StateProvider:` block with two new cases:

```go
case StateSearch:
    rows = append(rows, headerStyle.Render("ORCAI  New Session"))

    // Search input row
    inputStyle := lipgloss.NewStyle().Foreground(styles.Comment).Width(w).Padding(0, 1)
    rows = append(rows, inputStyle.Render(m.searchInput.View()))

    if len(m.filteredItems) == 0 {
        rows = append(rows, inactiveStyle.Render("  no matches"))
    } else {
        lastKind := ""
        for i, item := range m.filteredItems {
            // Insert group header when the kind changes.
            if item.Kind != lastKind {
                groupLabel := item.Kind + "s" // "sessions", "pipelines", "skills", "agents", "providers"
                rows = append(rows, sectionStyle.Render("── "+groupLabel+" ──"))
                lastKind = item.Kind
            }

            nameStr := renderMatchHighlights(item.Name, item.matchIndexes)
            var suffix string
            if item.SourceTag != "" {
                suffix = "  " + item.SourceTag
            } else if item.Description != "" && item.Kind == "provider" {
                suffix = " ›"
            }

            if i == m.itemCursor {
                rows = append(rows, activeStyle.Render("▎ "+nameStr+suffix))
            } else {
                rows = append(rows, inactiveStyle.Render("  "+nameStr+suffix))
            }
        }
    }
    rows = append(rows, footerStyle.Render("↑↓ nav  enter select  type to search"))

case StateProvider: // pick which CLI to use for the selected skill/agent
    title := "ORCAI  Select Provider"
    if m.selectedItem != nil {
        title = "ORCAI  Launch: " + m.selectedItem.Name
    }
    rows = append(rows, headerStyle.Render(title))
    for i, p := range m.skillProviders {
        if i == m.spCursor {
            rows = append(rows, activeStyle.Render("▎ "+p.Label))
        } else {
            rows = append(rows, inactiveStyle.Render("  "+p.Label))
        }
    }
    rows = append(rows, footerStyle.Render("↑↓ nav  enter select  esc back"))
```

**Step 3: Build and check for compilation errors**

```bash
go build ./internal/picker/...
```

Expected: no errors. If `strings` import is missing, add it.

**Step 4: Run all tests**

```bash
go test ./internal/picker/... -v
```

Expected: all PASS.

**Step 5: Commit**

```bash
git add internal/picker/picker.go
git commit -m "feat(picker): StateSearch fuzzy list view with group headers and match highlighting"
```

---

### Task 6: Pipeline and skill/agent launch — doLaunch + inject text in Run()

**Files:**
- Modify: `internal/picker/picker.go` (`doLaunch`, `Run`)

**Step 1: Update doLaunch to handle pipeline and skill/agent launches**

Find `doLaunch()` and replace it entirely:

```go
// doLaunch performs the session launch from pickerModel state.
// For existing sessions, this is a no-op (focus already handled in Update).
// For pipelines, opens a new tmux window running `orcai pipeline run <name>`.
// For skill/agent and raw provider sessions, calls launchFrom.
func (m *pickerModel) doLaunch() {
	if m.selectedSession != nil {
		return // existing session already focused in Update
	}

	basePath := strings.TrimSpace(m.workdirInput.Value())
	if basePath == "" {
		basePath = currentPanePath()
	}

	// Pipeline: launch via the orcai pipeline subcommand in a new tmux window.
	if m.selectedItem != nil && m.selectedItem.Kind == "pipeline" {
		name := m.selectedItem.Name
		windowName := "pipeline-" + name
		self, err := os.Executable()
		if err != nil {
			return
		}
		args := []string{"new-window", "-t", "orcai", "-n", windowName, self, "pipeline", "run", name}
		exec.Command("tmux", args...).Run() //nolint:errcheck
		return
	}

	// Skill/agent and raw provider: launch the selected CLI in a new tmux window.
	m.launchedWorktree = launchFrom(m.selectedProvider, m.selectedModelID, basePath)
}
```

**Step 2: Add sendInjectText helper**

Add this function near the bottom of `picker.go`, before `Run()`:

```go
// sendInjectText waits for the newly launched CLI to start, then sends the
// inject text (e.g. "/golang-patterns" or "@beast-mode ") to the active
// tmux window. The 2-second delay mirrors the opsx.ProviderSend pattern.
func sendInjectText(injectText string) {
	if injectText == "" {
		return
	}
	time.Sleep(2 * time.Second)
	exec.Command("tmux", "send-keys", "-t", "orcai", injectText, "Enter").Run() //nolint:errcheck
}
```

Make sure `"time"` is in the import block.

**Step 3: Update Run() to call sendInjectText for skill/agent launches**

Find the existing `Run()` function:

```go
func Run() {
    p := tea.NewProgram(newPickerModel(), tea.WithAltScreen())
    result, err := p.Run()
    if err != nil {
        fmt.Printf("picker error: %v\n", err)
        return
    }
    if pm, ok := result.(pickerModel); ok && pm.openspecFeature != "" {
        opsx.ProviderSend(pm.openspecFeature, pm.selectedProvider.ID, pm.launchedWorktree)
    }
}
```

Replace with:

```go
func Run() {
	p := tea.NewProgram(newPickerModel(), tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		fmt.Printf("picker error: %v\n", err)
		return
	}
	pm, ok := result.(pickerModel)
	if !ok {
		return
	}
	// OpenSpec workflow takes priority — it includes its own send delay.
	if pm.openspecFeature != "" {
		opsx.ProviderSend(pm.openspecFeature, pm.selectedProvider.ID, pm.launchedWorktree)
		return
	}
	// Skill/agent launch: inject the slash command or @mention after CLI starts.
	if pm.selectedItem != nil && pm.selectedItem.InjectText != "" {
		sendInjectText(pm.selectedItem.InjectText)
	}
}
```

**Step 4: Build everything**

```bash
go build ./...
```

Expected: clean build.

**Step 5: Run full test suite**

```bash
go test ./... 2>&1 | tail -20
```

Expected: all PASS.

**Step 6: Smoke test the binary**

```bash
./bin/orcai --help
```

Expected: help text prints without errors.

**Step 7: Commit**

```bash
git add internal/picker/picker.go
git commit -m "feat(picker): pipeline + skill/agent launch with inject text via tmux send-keys"
```

---

## Summary

| Task | What it builds |
|------|---------------|
| 1 | Add `sahilm/fuzzy` dependency |
| 2 | `PickerItem`, `ApplyFuzzy`, `BuildPickerItems` in `items.go` |
| 3 | `StateSearch` constant + test update |
| 4 | `pickerModel` fields + `Update` state machine |
| 5 | `View()` — fuzzy list with group headers and match highlights |
| 6 | `doLaunch` pipeline/skill support + `sendInjectText` + `Run()` wiring |
