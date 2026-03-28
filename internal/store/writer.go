package store

import (
	"database/sql"
	"time"
)

// writeOp is a single serialized write operation sent through the writer channel.
type writeOp struct {
	fn  func(*sql.DB) error
	res chan error
}

// writer serializes all writes to the database through a single goroutine,
// preventing SQLite write contention.
type writer struct {
	ch   chan writeOp
	done chan struct{}
}

// newWriter starts the writer goroutine and returns a writer.
func newWriter(db *sql.DB) *writer {
	w := &writer{
		ch:   make(chan writeOp, 64),
		done: make(chan struct{}),
	}
	go w.loop(db)
	return w
}

// loop processes write operations sequentially.
func (w *writer) loop(db *sql.DB) {
	defer close(w.done)
	for op := range w.ch {
		op.res <- op.fn(db)
	}
}

// send sends fn to the writer goroutine and blocks until it completes.
func (w *writer) send(fn func(*sql.DB) error) error {
	res := make(chan error, 1)
	w.ch <- writeOp{fn: fn, res: res}
	return <-res
}

// close stops the writer goroutine gracefully.
func (w *writer) close() {
	close(w.ch)
	<-w.done
}

// RecordRunStart inserts a new in-flight run row and returns its ID.
// started_at is recorded in unix milliseconds.
// metadata is an optional JSON blob (pass "" to omit).
func (s *Store) RecordRunStart(kind, name, metadata string) (int64, error) {
	startedAt := time.Now().UnixMilli()
	var id int64
	err := s.writer.send(func(db *sql.DB) error {
		res, err := db.Exec(
			`INSERT INTO runs (kind, name, started_at, metadata) VALUES (?, ?, ?, ?)`,
			kind, name, startedAt, metadata,
		)
		if err != nil {
			return err
		}
		id, err = res.LastInsertId()
		return err
	})
	return id, err
}

// RecordRunComplete updates the run row with exit status, stdout, stderr,
// and calls AutoPrune with the configured retention settings.
func (s *Store) RecordRunComplete(id int64, exitStatus int, stdout, stderr string) error {
	finishedAt := time.Now().UnixMilli()
	return s.writer.send(func(db *sql.DB) error {
		_, err := db.Exec(
			`UPDATE runs
			    SET finished_at = ?,
			        exit_status = ?,
			        stdout      = ?,
			        stderr      = ?
			  WHERE id = ?`,
			finishedAt, exitStatus, stdout, stderr, id,
		)
		if err != nil {
			return err
		}
		return autoPruneDB(db, s.cfg.MaxAgeDays, s.cfg.MaxRows)
	})
}
