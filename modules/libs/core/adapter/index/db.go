// Package index is the cache the notes are queried from. The vaults on disk are
// what is true; this is what makes them fast to ask questions of.
//
// It owns the connection and the schema, and hands out the repositories that
// use them. Each aggregate lives in its own package beside its own SQL, so
// adding one is a new folder.
package index

import (
	"context"
	"database/sql"
	"net/url"
	"slices"
	"strings"

	_ "modernc.org/sqlite"
	_ "modernc.org/sqlite/vec"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/chunk"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/note"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/vault"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testonly"
)

// DB owns the connections.
//
// There are two pools, because SQLite has one writer and any number of readers.
// The write pool is capped at a single connection, so writers queue in Go,
// where waiting is cheap and ordered. The read pool is unrestricted: in WAL
// mode a reader never waits for the writer, which is what lets a search answer
// while a scan is still running.
//
// Several processes open the one file. Their writers queue in SQLite, under the
// busy timeout, and a writer still waiting when it runs out says so.
type DB struct {
	write *sql.DB
	read  *sql.DB
}

// pragmas are carried in the connection string, so every connection in the
// pool opens with them. `sql.Open` returns a pool, and a PRAGMA statement
// configures whichever single connection happened to serve it. Foreign keys and
// the busy timeout are per-connection state, so a statement-based setup leaves
// every other connection with foreign keys off — and no ordinary test can see
// it, because a sequential test keeps being handed the one connection that was
// configured.
var pragmas = []string{
	// WAL so a long scan does not block readers. This one is persisted in the
	// database header, and belongs with the rest all the same.
	"journal_mode(WAL)",
	// Foreign keys so removing a vault cannot leave rows pointing at nothing.
	"foreign_keys(1)",
	// Wait for a writer, up to five seconds, before SQLITE_BUSY. The writer
	// waited for may be in another process.
	"busy_timeout(5000)",
	"cache_size(-65536)",
}

const (
	shipped        = "synchronous(NORMAL)"
	unsynchronised = "synchronous(OFF)"
)

// synchronous is what a commit waits for. The index is a cache: a crash costs a
// rescan, never data, and NORMAL under WAL is what that is worth.
var synchronous = shipped

// Unsynchronised stops every index this process opens from here on waiting for
// the disk. WAL, the single writer and the busy timeout stay as they are; only
// the moment of the flush moves. The grant is obtainable only inside this
// module.
func Unsynchronised(testonly.Grant) { synchronous = unsynchronised }

// AsShipped puts back the setting a person's index runs with, for a measurement
// that has to pay what the application pays.
func AsShipped(testonly.Grant) { synchronous = shipped }

func Open(ctx context.Context, path string) (*DB, error) {
	write, err := sql.Open("sqlite", writeDSN(path))
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
func (d *DB) Chunks() *chunk.Repository { return chunk.NewRepository(d.write) }

// Maintenance writes, so it takes the pool that is allowed to.
func (d *DB) Maintenance() DatabaseMaintenance { return DatabaseMaintenance{d.write} }

// NoteQueries reads, so it takes the pool that does not wait for the writer.
func (d *DB) NoteQueries() *note.Queries { return note.NewQueries(d.read) }

// ChunkQueries reads, so a search answers while a scan is still writing.
func (d *DB) ChunkQueries() *chunk.Queries { return chunk.NewQueries(d.read) }

// Sources is the source and vector ports over the chunk tables. It writes and
// reads both, through the pool each half belongs to.
func (d *DB) Sources() sources {
	return sources{queries: queries{read: d.ChunkQueries()}, write: d.Chunks()}
}

func dsn(path string) string { return dsnOf(path, append(slices.Clone(pragmas), synchronous)) }

// writeDSN is what the write pool opens with. Every transaction on it takes the
// write lock at BEGIN, so one that reads before it writes waits its turn under
// the busy timeout.
func writeDSN(path string) string { return dsn(path) + "&_txlock=immediate" }

func dsnOf(path string, pragmas []string) string {
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
