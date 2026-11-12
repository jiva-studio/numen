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
	// This is the number of attempts a write makes to take the lock. The driver
	// waits the busy timeout on each attempt, so the total wait is that timeout
	// multiplied by this count, bounded by the caller's context.
	asks = 3
	// between is the pause before the next attempt. The write connection returns
	// to the pool during the pause, so other writes in this process are not
	// queued behind a writer that is only waiting.
	between = 100 * time.Millisecond
)

// Begin starts a write transaction, asking again while another process holds
// the write lock.
// The write pool opens with _txlock=immediate, so the lock is taken at BEGIN;
// a transaction that could not take it has done nothing and the retry is a
// new Begin.
func Begin(ctx context.Context, db *sql.DB) (*Transaction, error) {
	var tx *sql.Tx
	err := retry(ctx, func() error {
		var err error
		tx, err = db.BeginTx(ctx, nil)
		return err
	})
	if err != nil {
		// The driver interrupts a BEGIN whose context ends and reports the interrupt, not the cause.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, errors.Join(ctxErr, err)
		}
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
// Each statement is prepared on first use and reused, because the driver
// prepares afresh for every Exec and one transaction runs the same statements
// hundreds of times.
// A prepared statement is closed with its transaction, on commit and rollback alike.
type Transaction struct {
	*sql.Tx
	ready map[string]*sql.Stmt
}

// ExecContext, QueryContext and QueryRowContext start a statement only while
// ctx is live and then run it to its end. See Uninterrupted.
func (t *Transaction) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	ctx, err := Uninterrupted(ctx)
	if err != nil {
		return nil, err
	}
	return useStatement(ctx, t, query, func(prepared *sql.Stmt) (sql.Result, error) {
		return prepared.ExecContext(ctx, args...)
	})
}

func (t *Transaction) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	ctx, err := Uninterrupted(ctx)
	if err != nil {
		return nil, err
	}
	return useStatement(ctx, t, query, func(prepared *sql.Stmt) (*sql.Rows, error) {
		return prepared.QueryContext(ctx, args...)
	})
}

// QueryRowContext passes a statement that could not be prepared to the
// transaction itself, so the error is returned to the caller: a row carries the error
// and there is no other channel for it.
func (t *Transaction) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	ctx, err := Uninterrupted(ctx)
	if err != nil {
		return t.Tx.QueryRowContext(ctx, query, args...)
	}
	row, err := useStatement(ctx, t, query, func(prepared *sql.Stmt) (*sql.Row, error) {
		return prepared.QueryRowContext(ctx, args...), nil
	})
	if err != nil {
		return t.Tx.QueryRowContext(ctx, query, args...)
	}
	return row
}

// Uninterrupted returns ctx with its error when ctx is done, otherwise a context not canceled while the statement runs.
// The driver answers cancellation by interrupting the connection, and an
// interrupted prepare in the vec0 table leaves a nil statement that it then
// clears, which ends the process. A write transaction still rolls back on
// cancellation, between statements.
func Uninterrupted(ctx context.Context) (context.Context, error) {
	if err := ctx.Err(); err != nil {
		return ctx, err
	}
	return context.WithoutCancel(ctx), nil
}

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
// write lock. A statement outside a transaction is its own transaction, so a
// statement that reports the lock has made no change and runs again.
func Exec(ctx context.Context, db *sql.DB, query string, args ...any) (sql.Result, error) {
	var out sql.Result
	err := retry(ctx, func() error {
		var err error
		out, err = db.ExecContext(ctx, query, args...)
		return err
	})
	return out, err
}

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
