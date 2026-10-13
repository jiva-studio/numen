// Package sqlwrite runs writes on the index database and retries them while another process holds the lock.
package sqlwrite

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
func Begin(ctx context.Context, db *sql.DB) (*Transaction, error) {
	var tx *sql.Tx
	err := retry(ctx, func() error {
		var err error
		tx, err = db.BeginTx(ctx, nil)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &Transaction{Tx: tx}, nil
}

// Transaction is a write transaction.
type Transaction struct {
	*sql.Tx
}

func (t *Transaction) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	var out sql.Result
	err := prepare(ctx, t.Tx, query, func(stmt *sql.Stmt) error {
		var err error
		out, err = stmt.ExecContext(ctx, args...)
		return err
	})
	return out, err
}

func (t *Transaction) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	var out *sql.Rows
	err := prepare(ctx, t.Tx, query, func(stmt *sql.Stmt) error {
		var err error
		out, err = stmt.QueryContext(ctx, args...)
		return err
	})
	return out, err
}

// QueryRowContext hands a statement that could not be prepared to the
// transaction itself, which is where the reason reaches the caller: a row
// carries what went wrong and there is no other way to put it there.
func (t *Transaction) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	var out *sql.Row
	err := prepare(ctx, t.Tx, query, func(stmt *sql.Stmt) error {
		out = stmt.QueryRowContext(ctx, args...)
		return nil
	})
	if err != nil {
		return t.Tx.QueryRowContext(ctx, query, args...)
	}
	return out
}

// prepare runs a statement prepared on the transaction and closes it when done.
func prepare(ctx context.Context, tx *sql.Tx, query string, run func(*sql.Stmt) error) error {
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()
	return run(stmt)
}

// Exec runs one write statement, asking again while another process holds the
// write lock. A statement of its own is its own transaction, so one that was
// refused the lock wrote nothing.
func Exec(ctx context.Context, db *sql.DB, query string, args ...any) (sql.Result, error) {
	var out sql.Result
	err := retry(ctx, func() error {
		var err error
		out, err = db.ExecContext(ctx, query, args...)
		return err
	})
	return out, err
}

// retry runs a write, and runs it retry where the answer was that the database
// is locked. It is only for a write that has done nothing when it says so.
func retry(ctx context.Context, write func() error) error {
	var err error
	for ask := range asks {
		if ask > 0 {
			select {
			case <-time.After(between):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		if err = write(); !isLocked(err) {
			return err
		}
	}
	return err
}

// isLocked says the database was held by a writer this one waited out. The
// extended result codes carry the primary one in their low byte.
func isLocked(err error) bool {
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
