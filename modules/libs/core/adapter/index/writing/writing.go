// Package writing is how a write on the index waits out a writer in another
// process.
//
// SQLite has one write lock for the whole file, and several processes open the
// one index. A driver that has waited out its busy timeout answers that the
// database is locked, and a write that meets that answer asks again.
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
func Begin(ctx context.Context, db *sql.DB) (*Transaction, error) {
	var tx *sql.Tx
	err := again(ctx, func() error {
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

// QueryRowContext hands a statement that could not be prepared to the
// transaction itself, which is where the reason reaches the caller: a row
// carries what went wrong and there is no other way to put it there.
func (t *Transaction) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	//nolint:sqlclosecheck // the statement is the transaction's and is closed with it
	prepared, err := t.prepare(ctx, query)
	if err != nil {
		return t.Tx.QueryRowContext(ctx, query, args...)
	}
	return prepared.QueryRowContext(ctx, args...)
}

// prepare is the statement this text was prepared as, preparing it the first
// time it is asked for. One transaction is written by one goroutine, which is
// what the pool of one write connection leaves it.
func (t *Transaction) prepare(ctx context.Context, query string) (*sql.Stmt, error) {
	if held, ok := t.ready[query]; ok {
		return held, nil
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
