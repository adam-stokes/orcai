package store

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// openTestStore creates a Store backed by a temporary directory.
func openTestStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := OpenAt(path)
	if err != nil {
		t.Fatalf("OpenAt: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestRecordRunStart(t *testing.T) {
	s := openTestStore(t)

	id, err := s.RecordRunStart("pipeline", "my-pipeline", "")
	if err != nil {
		t.Fatalf("RecordRunStart: %v", err)
	}
	if id <= 0 {
		t.Errorf("want id > 0, got %d", id)
	}
}

func TestRecordRunStart_WithMetadata(t *testing.T) {
	s := openTestStore(t)

	id, err := s.RecordRunStart("pipeline", "meta-test", `{"cwd":"/tmp","pipeline_file":"/tmp/foo.yaml"}`)
	if err != nil {
		t.Fatalf("RecordRunStart: %v", err)
	}

	runs, err := s.QueryRuns(1)
	if err != nil {
		t.Fatalf("QueryRuns: %v", err)
	}
	if len(runs) == 0 {
		t.Fatal("want 1 run, got 0")
	}
	if runs[0].ID != id {
		t.Errorf("want id %d, got %d", id, runs[0].ID)
	}
	if runs[0].Metadata != `{"cwd":"/tmp","pipeline_file":"/tmp/foo.yaml"}` {
		t.Errorf("want metadata blob, got %q", runs[0].Metadata)
	}
}

func TestRecordRunComplete(t *testing.T) {
	s := openTestStore(t)

	id, err := s.RecordRunStart("agent", "my-agent", "")
	if err != nil {
		t.Fatalf("RecordRunStart: %v", err)
	}

	if err := s.RecordRunComplete(id, 0, "hello stdout", "hello stderr"); err != nil {
		t.Fatalf("RecordRunComplete: %v", err)
	}

	runs, err := s.QueryRuns(10)
	if err != nil {
		t.Fatalf("QueryRuns: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("want 1 run, got %d", len(runs))
	}
	r := runs[0]
	if r.ID != id {
		t.Errorf("want id %d, got %d", id, r.ID)
	}
	if r.ExitStatus == nil {
		t.Fatal("want exit_status set, got nil")
	}
	if *r.ExitStatus != 0 {
		t.Errorf("want exit_status 0, got %d", *r.ExitStatus)
	}
	if r.Stdout != "hello stdout" {
		t.Errorf("want stdout 'hello stdout', got %q", r.Stdout)
	}
	if r.Stderr != "hello stderr" {
		t.Errorf("want stderr 'hello stderr', got %q", r.Stderr)
	}
	if r.FinishedAt == nil {
		t.Error("want finished_at set, got nil")
	}
}

func TestQueryRuns(t *testing.T) {
	s := openTestStore(t)

	// Insert three runs with distinct timestamps.
	names := []string{"first", "second", "third"}
	for _, n := range names {
		id, err := s.RecordRunStart("pipeline", n, "")
		if err != nil {
			t.Fatalf("RecordRunStart(%s): %v", n, err)
		}
		if err := s.RecordRunComplete(id, 0, "", ""); err != nil {
			t.Fatalf("RecordRunComplete(%s): %v", n, err)
		}
		// Small sleep to ensure distinct started_at values.
		time.Sleep(2 * time.Millisecond)
	}

	runs, err := s.QueryRuns(10)
	if err != nil {
		t.Fatalf("QueryRuns: %v", err)
	}
	if len(runs) != 3 {
		t.Fatalf("want 3 runs, got %d", len(runs))
	}

	// Verify descending order: third should come first.
	if runs[0].Name != "third" {
		t.Errorf("want first result 'third', got %q", runs[0].Name)
	}
	if runs[2].Name != "first" {
		t.Errorf("want last result 'first', got %q", runs[2].Name)
	}

	// Verify limit is respected.
	limited, err := s.QueryRuns(2)
	if err != nil {
		t.Fatalf("QueryRuns(limit=2): %v", err)
	}
	if len(limited) != 2 {
		t.Errorf("want 2 runs with limit=2, got %d", len(limited))
	}
}

func TestDeleteRun(t *testing.T) {
	s := openTestStore(t)

	id, err := s.RecordRunStart("pipeline", "to-delete", "")
	if err != nil {
		t.Fatalf("RecordRunStart: %v", err)
	}

	if err := s.DeleteRun(id); err != nil {
		t.Fatalf("DeleteRun: %v", err)
	}

	runs, err := s.QueryRuns(10)
	if err != nil {
		t.Fatalf("QueryRuns: %v", err)
	}
	for _, r := range runs {
		if r.ID == id {
			t.Errorf("run %d still present after DeleteRun", id)
		}
	}
}

func TestAutoPrune_ByAge(t *testing.T) {
	s := openTestStore(t)

	// Insert a run directly with a very old started_at (100 days ago in millis).
	// Since we are in the same package, we can access s.db directly.
	oldMillis := time.Now().Add(-100 * 24 * time.Hour).UnixMilli()
	_, err := s.db.Exec(
		`INSERT INTO runs (kind, name, started_at) VALUES ('pipeline', 'old-run', ?)`,
		oldMillis,
	)
	if err != nil {
		t.Fatalf("insert old run: %v", err)
	}

	// Insert a fresh run.
	_, freshErr := s.RecordRunStart("pipeline", "fresh-run", "")
	if freshErr != nil {
		t.Fatalf("RecordRunStart fresh: %v", freshErr)
	}

	// Prune with maxAgeDays=30 — the old run (100 days ago) should be removed.
	if err := s.AutoPrune(30, 10000); err != nil {
		t.Fatalf("AutoPrune: %v", err)
	}

	runs, err := s.QueryRuns(100)
	if err != nil {
		t.Fatalf("QueryRuns: %v", err)
	}
	for _, r := range runs {
		if r.Name == "old-run" {
			t.Error("old-run should have been pruned but still exists")
		}
	}
	found := false
	for _, r := range runs {
		if r.Name == "fresh-run" {
			found = true
		}
	}
	if !found {
		t.Error("fresh-run should still exist after pruning")
	}
}

func TestAutoPrune_ByCount(t *testing.T) {
	s := openTestStore(t)

	// Insert 15 runs.
	for i := 0; i < 15; i++ {
		id, err := s.RecordRunStart("pipeline", fmt.Sprintf("run-%02d", i), "")
		if err != nil {
			t.Fatalf("RecordRunStart run-%02d: %v", i, err)
		}
		if err := s.RecordRunComplete(id, 0, "", ""); err != nil {
			t.Fatalf("RecordRunComplete run-%02d: %v", i, err)
		}
	}

	// Prune with maxRows=10 — oldest 5 should be removed.
	if err := s.AutoPrune(3650, 10); err != nil {
		t.Fatalf("AutoPrune: %v", err)
	}

	runs, err := s.QueryRuns(100)
	if err != nil {
		t.Fatalf("QueryRuns: %v", err)
	}
	if len(runs) != 10 {
		t.Errorf("want 10 runs after prune, got %d", len(runs))
	}
}

func TestWALMode(t *testing.T) {
	s := openTestStore(t)

	var mode string
	if err := s.db.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Errorf("want journal_mode=wal, got %q", mode)
	}
}

func TestConcurrentWrites(t *testing.T) {
	s := openTestStore(t)

	const goroutines = 10
	errs := make([]error, goroutines)
	ids := make([]int64, goroutines)

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			id, err := s.RecordRunStart("agent", fmt.Sprintf("concurrent-%d", i), "")
			ids[i] = id
			errs[i] = err
		}()
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d: RecordRunStart error: %v", i, err)
		}
		if ids[i] <= 0 {
			t.Errorf("goroutine %d: want id > 0, got %d", i, ids[i])
		}
	}

	runs, err := s.QueryRuns(goroutines + 10)
	if err != nil {
		t.Fatalf("QueryRuns: %v", err)
	}
	if len(runs) != goroutines {
		t.Errorf("want %d runs, got %d", goroutines, len(runs))
	}
}
