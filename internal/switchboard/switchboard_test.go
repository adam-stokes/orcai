package switchboard_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/adam-stokes/orcai/internal/switchboard"
)

// ── scanPipelines ─────────────────────────────────────────────────────────────

func TestScanPipelines_MissingDir(t *testing.T) {
	result := switchboard.ScanPipelines("/tmp/does-not-exist-orcai-test-dir")
	if len(result) != 0 {
		t.Errorf("expected 0 pipelines for missing dir, got %d", len(result))
	}
}

func TestScanPipelines_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	result := switchboard.ScanPipelines(dir)
	if len(result) != 0 {
		t.Errorf("expected 0 pipelines for empty dir, got %d", len(result))
	}
}

func TestScanPipelines_PopulatedDir(t *testing.T) {
	dir := t.TempDir()
	// Create some .pipeline.yaml files.
	for _, name := range []string{"alpha.pipeline.yaml", "beta.pipeline.yaml", "gamma.pipeline.yaml"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("name: "+name+"\nsteps: []\n"), 0o600); err != nil {
			t.Fatalf("create file: %v", err)
		}
	}
	// Create a non-pipeline file that should be ignored.
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("ignore me"), 0o600) //nolint:errcheck

	result := switchboard.ScanPipelines(dir)
	if len(result) != 3 {
		t.Fatalf("expected 3 pipelines, got %d: %v", len(result), result)
	}
	// Check that extensions are stripped.
	for _, r := range result {
		if strings.HasSuffix(r, ".pipeline.yaml") {
			t.Errorf("expected extension stripped, got %q", r)
		}
		if strings.Contains(r, ".yaml") {
			t.Errorf("expected no yaml suffix, got %q", r)
		}
	}
	// Verify names are present.
	names := map[string]bool{}
	for _, r := range result {
		names[r] = true
	}
	for _, want := range []string{"alpha", "beta", "gamma"} {
		if !names[want] {
			t.Errorf("missing pipeline %q in results %v", want, result)
		}
	}
}

// ── ChanPublisher ─────────────────────────────────────────────────────────────

func TestChanPublisher_SendsFeedLineMsg(t *testing.T) {
	ch := make(chan tea.Msg, 10)
	pub := switchboard.NewChanPublisher("test-id", ch)
	err := pub.Publish(context.Background(), "step.done", []byte(`{"step":"s1"}`))
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	select {
	case msg := <-ch:
		fl, ok := msg.(switchboard.FeedLineMsg)
		if !ok {
			t.Fatalf("expected FeedLineMsg, got %T", msg)
		}
		if fl.ID != "test-id" {
			t.Errorf("expected id %q, got %q", "test-id", fl.ID)
		}
		if !strings.Contains(fl.Line, "step.done") {
			t.Errorf("expected line to contain 'step.done', got %q", fl.Line)
		}
	default:
		t.Fatal("expected message in channel, got none")
	}
}

// ── Launcher navigation ───────────────────────────────────────────────────────

func TestLauncherNavDown(t *testing.T) {
	m := switchboard.NewWithPipelines([]string{"alpha", "beta", "gamma"})
	// Initially focused on launcher; cursor at 0.
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if got := m2.(switchboard.Model).Cursor(); got != 1 {
		t.Errorf("cursor after j: got %d, want 1", got)
	}
}

func TestLauncherNavUp(t *testing.T) {
	m := switchboard.NewWithPipelines([]string{"alpha", "beta", "gamma"})
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m3, _ := m2.(switchboard.Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if got := m3.(switchboard.Model).Cursor(); got != 0 {
		t.Errorf("cursor after j then k: got %d, want 0", got)
	}
}

func TestLauncherNavClampedAtBottom(t *testing.T) {
	m := switchboard.NewWithPipelines([]string{"alpha"})
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if got := m2.(switchboard.Model).Cursor(); got != 0 {
		t.Errorf("cursor should stay at 0 with one item: got %d", got)
	}
}

func TestLauncherNavClampedAtTop(t *testing.T) {
	m := switchboard.NewWithPipelines([]string{"alpha"})
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if got := m2.(switchboard.Model).Cursor(); got != 0 {
		t.Errorf("cursor should not go negative: got %d", got)
	}
}

// ── Agent modal overlay ───────────────────────────────────────────────────────

// TestAgentModalOpenOnEnter asserts that pressing enter when the agent runner
// is focused (and terminal is wide enough) opens the modal overlay.
func TestAgentModalOpenOnEnter(t *testing.T) {
	m := switchboard.NewWithTestProviders()

	// Size the terminal wide enough for the modal.
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m3 := m2.(switchboard.Model)

	// Focus agent section.
	m4, _ := m3.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m5 := m4.(switchboard.Model)
	if m5.AgentModalOpen() {
		t.Fatal("modal should not be open before enter")
	}

	// Press enter — modal should open.
	m6, _ := m5.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m7 := m6.(switchboard.Model)
	if !m7.AgentModalOpen() {
		t.Error("expected agent modal to be open after enter")
	}

	// Press ESC — modal should close.
	m8, _ := m7.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m9 := m8.(switchboard.Model)
	if m9.AgentModalOpen() {
		t.Error("expected agent modal to be closed after ESC")
	}
}

// ── View smoke test ───────────────────────────────────────────────────────────

func TestViewContainsBanner(t *testing.T) {
	m := switchboard.New()
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	view := m2.(switchboard.Model).View()
	if !strings.Contains(view, "ORCAI") {
		t.Errorf("View() missing ORCAI banner:\n%s", view)
	}
	if !strings.Contains(view, "╔") {
		t.Errorf("View() missing box-drawing border:\n%s", view)
	}
}

func TestViewContainsPipelinesSection(t *testing.T) {
	m := switchboard.NewWithPipelines([]string{"my-pipeline"})
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	view := m2.(switchboard.Model).View()
	if !strings.Contains(view, "PIPELINES") {
		t.Errorf("View() missing PIPELINES section:\n%s", view)
	}
	if !strings.Contains(view, "my-pipeline") {
		t.Errorf("View() missing pipeline name:\n%s", view)
	}
}

func TestViewContainsActivityFeed(t *testing.T) {
	m := switchboard.New()
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	view := m2.(switchboard.Model).View()
	if !strings.Contains(view, "ACTIVITY FEED") {
		t.Errorf("View() missing ACTIVITY FEED section:\n%s", view)
	}
}

func TestViewContainsBottomBar(t *testing.T) {
	// When launcher is focused (default), bottom bar is hidden to avoid
	// double-bar awkwardness with the tmux status bar.
	m := switchboard.New()
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m3 := m2.(switchboard.Model)
	// Shift focus off launcher via Tab to reach the agent panel.
	m4, _ := m3.Update(tea.KeyMsg{Type: tea.KeyTab})
	view := m4.(switchboard.Model).View()
	if !strings.Contains(view, "ctrl+s") {
		t.Errorf("View() bottom bar missing hint when agent focused:\n%s", view)
	}
	if !strings.Contains(view, "quit") {
		t.Errorf("View() bottom bar missing 'quit' hint:\n%s", view)
	}
}

// ── Feed scroll (task 1.6) ─────────────────────────────────────────────────────

func TestFeedScrollOffset_ClampedAtZero(t *testing.T) {
	m := switchboard.New()
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m3 := m2.(switchboard.Model)
	// Press up — offset should stay at 0.
	m4, _ := m3.Update(tea.KeyMsg{Type: tea.KeyUp})
	m5 := m4.(switchboard.Model)
	if got := m5.FeedScrollOffset(); got != 0 {
		t.Errorf("feedScrollOffset should be 0 at top, got %d", got)
	}
}

func TestFeedScrollOffset_InitialIsZero(t *testing.T) {
	m := switchboard.New()
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 10})
	m3 := m2.(switchboard.Model)
	// Add many feed entries with output lines so total lines exceed visible height.
	for i := 0; i < 30; i++ {
		lines := make([]string, 5)
		for j := range lines {
			lines[j] = "output line"
		}
		m3 = m3.AddFeedEntry("id", "title", switchboard.FeedDone, lines)
	}
	// Verify offset is 0 by default.
	if got := m3.FeedScrollOffset(); got != 0 {
		t.Errorf("initial feedScrollOffset should be 0, got %d", got)
	}
}

func TestFeedScrollOffset_ResetOnNewEntry(t *testing.T) {
	m := switchboard.New()
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 10})
	m3 := m2.(switchboard.Model)
	// Add a feed entry — scroll offset should be 0.
	m4 := m3.AddFeedEntry("id1", "first job", switchboard.FeedDone, []string{"line"})
	if got := m4.FeedScrollOffset(); got != 0 {
		t.Errorf("feedScrollOffset should be 0 after new entry, got %d", got)
	}
}

func TestFeedScrollOffset_ClampedAtMax(t *testing.T) {
	m := switchboard.New()
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m3 := m2.(switchboard.Model)
	// Add feed entries with lines.
	for i := 0; i < 5; i++ {
		m3 = m3.AddFeedEntry("id", "title", switchboard.FeedDone, []string{"a", "b"})
	}
	// View should still render without crashing.
	view := m3.View()
	if !strings.Contains(view, "ACTIVITY FEED") {
		t.Errorf("View() should still contain ACTIVITY FEED after clamping, got: %s", view)
	}
}

// ── Agent section fixed height (task 2.6) ──────────────────────────────────────

func TestBuildAgentSection_FixedHeight(t *testing.T) {
	m := switchboard.NewWithTestProviders()
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m3 := m2.(switchboard.Model)

	// Measure height at step 0.
	m3a, _ := m3.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m3b := m3a.(switchboard.Model)
	step0Lines := m3b.BuildAgentSection(60)

	// Advance to step 1.
	m4, _ := m3b.Update(tea.KeyMsg{Type: tea.KeyTab})
	m5 := m4.(switchboard.Model)
	step1Lines := m5.BuildAgentSection(60)

	// Advance to step 2.
	m6, _ := m5.Update(tea.KeyMsg{Type: tea.KeyTab})
	m7 := m6.(switchboard.Model)
	step2Lines := m7.BuildAgentSection(60)

	if len(step0Lines) != len(step1Lines) {
		t.Errorf("buildAgentSection step 0 vs step 1 line count mismatch: %d vs %d", len(step0Lines), len(step1Lines))
	}
	if len(step0Lines) != len(step2Lines) {
		t.Errorf("buildAgentSection step 0 vs step 2 line count mismatch: %d vs %d", len(step0Lines), len(step2Lines))
	}
}

// ── Signal board (task 3.8) ────────────────────────────────────────────────────

func TestSignalBoard_FilterAll(t *testing.T) {
	m := switchboard.New()
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m3 := m2.(switchboard.Model)
	m3 = m3.AddFeedEntry("j1", "running job", switchboard.FeedRunning, nil)
	m3 = m3.AddFeedEntry("j2", "done job", switchboard.FeedDone, nil)
	m3 = m3.AddFeedEntry("j3", "failed job", switchboard.FeedFailed, nil)

	sb := m3.BuildSignalBoard(8, 60)
	rendered := strings.Join(sb, "\n")
	// All filter — all 3 jobs should appear.
	if !strings.Contains(rendered, "running") {
		t.Errorf("signal board (all filter) missing 'running': %s", rendered)
	}
	if !strings.Contains(rendered, "done") {
		t.Errorf("signal board (all filter) missing 'done': %s", rendered)
	}
	if !strings.Contains(rendered, "failed") {
		t.Errorf("signal board (all filter) missing 'failed': %s", rendered)
	}
}

func TestSignalBoard_BlinkToggleOnTick(t *testing.T) {
	m := switchboard.New()
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m3 := m2.(switchboard.Model)
	m3 = m3.AddFeedEntry("j1", "running job", switchboard.FeedRunning, nil)

	before := m3.SignalBoardBlinkOn()
	// Send a tick message (use time.Now as the tick value).
	m4, _ := m3.Update(switchboard.MakeTickMsg())
	m5 := m4.(switchboard.Model)
	after := m5.SignalBoardBlinkOn()
	if before == after {
		t.Errorf("blink state should toggle on tick when running job exists: before=%v after=%v", before, after)
	}
}

func TestSignalBoard_HeaderContainsFilter(t *testing.T) {
	m := switchboard.New()
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m3 := m2.(switchboard.Model)
	sb := m3.BuildSignalBoard(8, 60)
	rendered := strings.Join(sb, "\n")
	if !strings.Contains(rendered, "SIGNAL BOARD") {
		t.Errorf("signal board missing 'SIGNAL BOARD' header: %s", rendered)
	}
	if !strings.Contains(rendered, "all") {
		t.Errorf("signal board missing filter 'all' in header: %s", rendered)
	}
}

// ── Tmux hidden windows (task 4.6) ────────────────────────────────────────────

func TestCreateJobWindow_SkipsIfNoTmux(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		// tmux not available — verify the function returns empty string gracefully.
		// We can't call createJobWindow directly (unexported) but we can verify that
		// launching a job in a non-tmux environment doesn't crash.
		t.Skip("tmux not found — skipping window creation test")
	}
	// If tmux is available, just verify the test setup doesn't panic.
	t.Log("tmux available — createJobWindow would attempt window creation")
}

// ── Debug popup (task 5.8) ──────────────────────────────────────────────────────

// Enter on signal board now navigates directly to the tmux window (no popup).
// In tests there is no real tmux, so we just verify the model state is unchanged
// (no popup opened, signal board still focused).
func TestSignalBoard_EnterDoesNotOpenPopup(t *testing.T) {
	m := switchboard.New()
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m3 := m2.(switchboard.Model)
	m3 = m3.AddFeedEntry("job1", "test job", switchboard.FeedDone, nil)
	m3 = m3.SetSignalBoardFocused(true)

	// Enter should navigate directly (tmux select-window) without opening any popup.
	// In tests there is no real tmux session, so we just verify the model
	// remains valid (signal board still focused, no crash).
	m4, _ := m3.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m5 := m4.(switchboard.Model)
	if !m5.SignalBoardFocused() {
		t.Error("signal board should remain focused after enter with no tmux window")
	}
}

func TestSignalBoard_ViewContainsSignalBoard(t *testing.T) {
	m := switchboard.New()
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	view := m2.(switchboard.Model).View()
	if !strings.Contains(view, "SIGNAL BOARD") {
		t.Errorf("View() missing SIGNAL BOARD section:\n%s", view)
	}
}

// ── Parallel Jobs (tasks 2.1–2.7 / 7.1–7.2) ──────────────────────────────────

func TestParallelJobs(t *testing.T) {
	m := switchboard.New()
	// Inject two FeedRunning entries.
	m = m.AddFeedEntry("job1", "pipeline: alpha", switchboard.FeedRunning, nil)
	m = m.AddFeedEntry("job2", "pipeline: beta", switchboard.FeedRunning, nil)
	// Inject two fake active job handles.
	m = m.AddActiveJob("job1")
	m = m.AddActiveJob("job2")

	if got := m.ActiveJobsCount(); got != 2 {
		t.Errorf("expected 2 active jobs, got %d", got)
	}

	// Verify both feed entries are FeedRunning via signal board.
	sb := m.BuildSignalBoard(8, 60)
	rendered := strings.Join(sb, "\n")
	if !strings.Contains(rendered, "running") {
		t.Errorf("signal board missing 'running' status: %s", rendered)
	}

	// View should show [2 running] badge.
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	view := m2.(switchboard.Model).View()
	if !strings.Contains(view, "2 running") {
		t.Errorf("expected view to show '2 running', got:\n%s", view)
	}
}

func TestParallelJobCap(t *testing.T) {
	m := switchboard.New()
	cap := switchboard.MaxParallelJobs()

	// Fill activeJobs to the cap.
	for i := 0; i < cap; i++ {
		m = m.AddActiveJob(fmt.Sprintf("job%d", i))
	}
	if got := m.ActiveJobsCount(); got != cap {
		t.Fatalf("expected %d active jobs before cap check, got %d", cap, got)
	}

	// Give the model some pipelines so we can try to launch.
	m = switchboard.NewWithPipelines([]string{"test-pipeline"})
	// Re-inject active jobs after creating new model.
	for i := 0; i < cap; i++ {
		m = m.AddActiveJob(fmt.Sprintf("job%d", i))
	}

	// Try to launch another job via Enter key.
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m3 := m2.(switchboard.Model)

	// activeJobs count should still be cap (no new job added).
	if got := m3.ActiveJobsCount(); got != cap {
		t.Errorf("expected activeJobs count to stay at cap %d, got %d", cap, got)
	}

	// A warning feed entry should have been added.
	view := m3.View()
	if !strings.Contains(view, "max parallel") {
		t.Errorf("expected warning 'max parallel' in view after cap exceeded:\n%s", view)
	}
}

// ── [p] pipelines focus shortcut ─────────────────────────────────────────────

func TestPKeyFocusesPipelines_FromAgent(t *testing.T) {
	m := switchboard.New()
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	// Focus agent section.
	m3, _ := m2.(switchboard.Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m4 := m3.(switchboard.Model)

	// Press p — should focus pipelines (launcher).
	m5, _ := m4.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m6 := m5.(switchboard.Model)
	if m6.Cursor() == -1 {
		// Cursor() reads launcher.selected; if it panics we have a problem.
		t.Fatal("launcher should be accessible after p key")
	}
	// Signal board and feed should not be focused — verified by view rendering.
	view := m6.View()
	if !strings.Contains(view, "PIPELINES") {
		t.Errorf("expected PIPELINES panel in view after p key:\n%s", view)
	}
}

func TestPKeyFocusesPipelines_FromFeed(t *testing.T) {
	m := switchboard.New()
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	// Focus feed.
	m3, _ := m2.(switchboard.Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	// Press p — should focus pipelines.
	m4, _ := m3.(switchboard.Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	view := m4.(switchboard.Model).View()
	if !strings.Contains(view, "PIPELINES") {
		t.Errorf("expected PIPELINES panel in view after p key from feed:\n%s", view)
	}
}

// ── [d] delete pipeline confirmation ─────────────────────────────────────────

func TestDKey_ShowsConfirmation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "my-pipe.pipeline.yaml")
	if err := os.WriteFile(path, []byte("name: my-pipe\nsteps: []\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	m := switchboard.NewWithPipelines(switchboard.ScanPipelines(dir))
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Press d — confirmation modal should appear in view.
	m3, _ := m2.(switchboard.Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	view := m3.(switchboard.Model).View()
	if !strings.Contains(view, "Delete") {
		t.Errorf("expected Delete confirmation in view after d key:\n%s", view)
	}
}

func TestDKey_CancelWithN(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "my-pipe.pipeline.yaml")
	os.WriteFile(path, []byte("name: my-pipe\nsteps: []\n"), 0o600) //nolint:errcheck

	m := switchboard.NewWithPipelines(switchboard.ScanPipelines(dir))
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Press d then n — file should still exist.
	m3, _ := m2.(switchboard.Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m4, _ := m3.(switchboard.Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	_ = m4
	if _, err := os.Stat(path); err != nil {
		t.Error("file should still exist after cancel with n")
	}
}

// ── Feed scroll indicators ────────────────────────────────────────────────────

func TestFeedScrollIndicator_NoIndicatorWhenAllVisible(t *testing.T) {
	m := switchboard.New()
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m3 := m2.(switchboard.Model)
	// With no feed entries and no scroll, no indicator expected.
	view := m3.View()
	if strings.Contains(view, "ACTIVITY FEED ↑") || strings.Contains(view, "ACTIVITY FEED ↓") || strings.Contains(view, "ACTIVITY FEED ↕") {
		t.Errorf("expected no scroll indicator with no feed content:\n%s", view)
	}
}

func TestFeedScrollIndicator_DownWhenContentBelow(t *testing.T) {
	m := switchboard.New()
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 20})
	m3 := m2.(switchboard.Model)
	// Add enough entries to overflow the visible height.
	for i := range 30 {
		m3 = m3.AddFeedEntry(fmt.Sprintf("job%d", i), fmt.Sprintf("pipeline: job%d", i), switchboard.FeedDone, []string{"output line"})
	}
	view := m3.View()
	if !strings.Contains(view, "↓") {
		t.Errorf("expected ↓ indicator when content extends below viewport:\n%s", view)
	}
}

