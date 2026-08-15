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

// DB owns the connections.
//
// There are two pools, because SQLite has one writer and any number of readers.
// The write pool is capped at a single connection so that writers queue in Go,
// where waiting is cheap and ordered, rather than in SQLite, where they compete
// for a lock and give up on a timeout. The read pool is unrestricted: in WAL
// mode a reader never waits for the writer, which is what lets a search answer
// while a scan is still running.
type DB struct {
	write *sql.DB
	read  *sql.DB
}

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
	// The index is a cache: a crash costs a rescan, never data. Paying an fsync
	// per commit to protect it buys nothing and dominates a rebuild.
	"synchronous(NORMAL)",
}

func Open(ctx context.Context, path string) (*DB, error) {
	write, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		return nil, err
	}
	// One writer. More connections would only mean more of them failing on a
	// busy lock.
	write.SetMaxOpenConns(1)

	if err := migrate(ctx, write); err != nil {
		write.Close()
		return nil, err
	}

	read, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		write.Close()
		return nil, err
	}
	return &DB{write: write, read: read}, nil
}

func (d *DB) Close() error {
	readErr := d.read.Close()
	if err := d.write.Close(); err != nil {
		return err
	}
	return readErr
}

func (d *DB) Vaults() *vault.Repository { return vault.NewRepository(d.write) }
func (d *DB) Notes() *note.Repository   { return note.NewRepository(d.write) }

// NoteQueries reads, so it takes the pool that does not wait for the writer.
func (d *DB) NoteQueries() *note.Queries { return note.NewQueries(d.read) }

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
