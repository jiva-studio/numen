// Package index is the cache the notes are queried from. The vaults on disk are
// what is true; this is what makes them fast to ask questions of.
//
// It owns the connection and the schema, and hands out the repositories that
// use them. Each aggregate lives in its own package beside its own SQL, so
// adding one is a new folder rather than more files in this one.
package index

import (
	"context"
	"database/sql"
	"net/url"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/index/note"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/index/vault"
)

// DB owns the connection. The repositories share one pool because they share
// one database — separating them is about what each may be asked to do, not
// about how many files there are.
type DB struct{ db *sql.DB }

// pragmas are carried in the connection string rather than executed after
// opening, because `sql.Open` returns a pool and executing a PRAGMA statement
// configures whichever single connection happened to serve it. Foreign keys and
// the busy timeout are per-connection state, so a statement-based setup leaves
// every other connection with foreign keys off — and no ordinary test can see
// it, because a sequential test keeps being handed the one connection that was
// configured.
var pragmas = []string{
	// WAL so a long scan does not block readers. This one is persisted in the
	// database header rather than per connection, but it belongs with the rest.
	"journal_mode(WAL)",
	// Foreign keys so removing a vault cannot leave rows pointing at nothing.
	"foreign_keys(1)",
	// Wait for a writer instead of failing immediately with SQLITE_BUSY.
	"busy_timeout(5000)",
}

func Open(ctx context.Context, path string) (*DB, error) {
	db, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		return nil, err
	}
	if err := migrate(ctx, db); err != nil {
		db.Close()
		return nil, err
	}
	return &DB{db: db}, nil
}

func (d *DB) Close() error { return d.db.Close() }

func (d *DB) Vaults() *vault.Repository  { return vault.NewRepository(d.db) }
func (d *DB) Notes() *note.Repository    { return note.NewRepository(d.db) }
func (d *DB) NoteQueries() *note.Queries { return note.NewQueries(d.db) }

func dsn(path string) string {
	q := url.Values{}
	for _, p := range pragmas {
		q.Add("_pragma", p)
	}
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	return path + separator + q.Encode()
}
