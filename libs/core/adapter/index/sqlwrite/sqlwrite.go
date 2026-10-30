// Package sqlwrite runs writes on the index database and retries them while another process holds the lock.
package sqlwrite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
	return &Transaction{Tx: tx, ready: map[string]*sql.Stmt{}}, nil
}

// InTransaction begins a write transaction, calls run, and commits it. A run
// that fails rolls the transaction back and returns its error, joined with the
// rollback error when the rollback itself failed.
func InTransaction(ctx context.Context, db *sql.DB, run func(tx *Transaction) error) error {
	tx, err := Begin(ctx, db)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	if err := run(tx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			return errors.Join(err, fmt.Errorf("rollback: %w", rollbackErr))
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// JoinClose returns err joined with closeErr, which is the error of closing a
// resource. It returns err itself when closeErr is nil.
func JoinClose(err, closeErr error) error {
	if closeErr == nil {
		return err
	}
	return errors.Join(err, closeErr)
}

// Transaction is a write transaction and the statements prepared inside it.
//
// SQLite reads the text of a statement each time one is prepared, and the
// driver prepares afresh for every Exec. A transaction that spans five hundred
// notes runs the same two dozen statements five hundred times over, so each of
// them is prepared here on its first use and run again after that.
//
// A prepared statement belongs to the transaction it was prepared in and is
// closed with it, on commit and on rollback alike.
type Transaction struct {
	*sql.Tx
	ready map[string]*sql.Stmt
}

func (t *Transaction) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return useStatement(ctx, t, query, func(prepared *sql.Stmt) (sql.Result, error) {
		return prepared.ExecContext(ctx, args...)
	})
}

func (t *Transaction) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return useStatement(ctx, t, query, func(prepared *sql.Stmt) (*sql.Rows, error) {
		return prepared.QueryContext(ctx, args...)
	})
}

// QueryRowContext hands a statement that could not be prepared to the
// transaction itself, which is where the reason reaches the caller: a row
// carries what went wrong and there is no other way to put it there.
func (t *Transaction) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	row, err := useStatement(ctx, t, query, func(prepared *sql.Stmt) (*sql.Row, error) {
		return prepared.QueryRowContext(ctx, args...), nil
	})
	if err != nil {
		return t.Tx.QueryRowContext(ctx, query, args...)
	}
	return row
}

// useStatement calls run with the statement this text was prepared as,
// preparing it the first time it is asked for. The transaction owns the
// statement and closes it on commit and on rollback. One transaction is written
// by one goroutine, which is what the pool of one write connection leaves it.
func useStatement[T any](
	ctx context.Context, t *Transaction, query string, run func(*sql.Stmt) (T, error),
) (T, error) {
	prepared, ok := t.ready[query]
	if !ok {
		var err error
		prepared, err = t.Tx.PrepareContext(ctx, query)
		if err != nil {
			var zero T
			return zero, err
		}
		t.ready[query] = prepared
	}
	return run(prepared)
}

// Exec runs one write statement, asking again while another process holds the
// write lock. A statement of its own is its own transaction, so one that was
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
	var response *sqlite.Error
	if !errors.As(err, &response) {
		return false
	}
	switch response.Code() & 0xff {
	case sqlite3.SQLITE_BUSY, sqlite3.SQLITE_LOCKED:
		return true
	}
	return false
}
