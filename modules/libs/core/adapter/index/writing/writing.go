// Package writing is how a write on the index waits out a writer in another
// process.
//
// SQLite has one write lock for the whole file. Inside this process the writers
// queue on the single write connection, where waiting is cheap and ordered; but
// several processes open the one index, and there the lock is the driver's to
// wait for. It waits the busy timeout and then answers that the database is
// locked.
//
// That answer is transient: the other writer commits and the lock is free. So a
// write that meets it asks again, rather than handing a person a scan that
// stopped or a note that was never indexed.
package writing

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

const (
	// asks is how many times a write asks for the lock. The driver waits the
	// busy timeout on each of them, so the wait is that timeout this many times
	// over, and no longer than the caller's context allows.
	asks = 3
	// between is the pause before asking again. The write connection is back in
	// the pool while it lasts, so whatever else this process wants to write is
	// not queued behind a writer that is only waiting.
	between = 100 * time.Millisecond
)

// Begin starts a write transaction, asking again while another process holds
// the write lock.
//
// The write pool opens with _txlock=immediate, so the lock is taken at BEGIN: a
// transaction that could not take it has done nothing, and beginning again is
// the whole of the retry.
func Begin(ctx context.Context, db *sql.DB) (*sql.Tx, error) {
	var tx *sql.Tx
	err := again(ctx, func() error {
		var err error
		tx, err = db.BeginTx(ctx, nil)
		return err
	})
	return tx, err
}

// Exec runs one write statement, asking again while another process holds the
// write lock. A statement of its own is its own transaction, so one that was
// refused the lock wrote nothing.
func Exec(ctx context.Context, db *sql.DB, query string, args ...any) (sql.Result, error) {
	var out sql.Result
	err := again(ctx, func() error {
		var err error
		out, err = db.ExecContext(ctx, query, args...)
		return err
	})
	return out, err
}

// again runs a write, and runs it again where the answer was that the database
// is locked. It is only for a write that has done nothing when it says so.
func again(ctx context.Context, write func() error) error {
	var err error
	for ask := range asks {
		if ask > 0 {
			select {
			case <-time.After(between):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		if err = write(); !locked(err) {
			return err
		}
	}
	return err
}

// locked says the database was held by a writer this one waited out. The
// extended result codes carry the primary one in their low byte.
func locked(err error) bool {
	var said *sqlite.Error
	if !errors.As(err, &said) {
		return false
	}
	switch said.Code() & 0xff {
	case sqlite3.SQLITE_BUSY, sqlite3.SQLITE_LOCKED:
		return true
	}
	return false
}
