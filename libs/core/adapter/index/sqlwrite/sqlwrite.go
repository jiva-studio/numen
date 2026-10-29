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
	//nolint:sqlclosecheck // the statement is the transaction's and is closed with it
	prepared, err := t.prepare(ctx, query)
	if err != nil {
		return nil, err
	}
	return prepared.ExecContext(ctx, args...)
}

func (t *Transaction) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	//nolint:sqlclosecheck // the statement is the transaction's and is closed with it
	prepared, err := t.prepare(ctx, query)
	if err != nil {
		return nil, err
	}
	return prepared.QueryContext(ctx, args...)
}

// QueryRowContext passes a statement that could not be prepared to the
// transaction itself, so the error is returned to the caller: a row carries the error
// and there is no other channel for it.
func (t *Transaction) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	//nolint:sqlclosecheck // the statement is the transaction's and is closed with it
	prepared, err := t.prepare(ctx, query)
	if err != nil {
		return t.Tx.QueryRowContext(ctx, query, args...)
	}
	return prepared.QueryRowContext(ctx, args...)
}

// prepare returns the prepared statement for this text, preparing it on first
// use. One transaction is written by one goroutine, because the pool has a
// single write connection.
func (t *Transaction) prepare(ctx context.Context, query string) (*sql.Stmt, error) {
	if target, ok := t.ready[query]; ok {
		return target, nil
	}
	prepared, err := t.Tx.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	t.ready[query] = prepared
	return prepared, nil
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

// retry runs a write again when the database is locked. It applies only to a
// write that has made no change when it reports the lock.
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

// isLocked reports whether the database was locked by another writer. The
// extended result codes carry the primary code in their low byte.
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
