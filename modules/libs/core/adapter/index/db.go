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
	"os"
	"strings"

	_ "modernc.org/sqlite"
	_ "modernc.org/sqlite/vec"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/chunk"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/note"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/vault"
)

// DB owns the connections.
//
// There are two pools, because SQLite has one writer and any number of readers.
// The write pool is capped at a single connection, so writers queue in Go,
// where waiting is cheap and ordered. The read pool is unrestricted: in WAL
// mode a reader never waits for the writer, which is what lets a search answer
// while a scan is still running.
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
	// Wait for a writer, up to five seconds, before SQLITE_BUSY.
	"busy_timeout(5000)",
	// The index is a cache: a crash costs a rescan, never data. Paying an fsync
	// per commit to protect it buys nothing and dominates a rebuild.
	"synchronous(NORMAL)",
	"cache_size(-65536)",
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

// Reading is the cache opened by a process that only asks it questions.
//
// It holds the read pool alone: there is nothing here to write with, so a
// binary that takes the index as it finds it cannot be the one that changes it.
type Reading struct {
	read *sql.DB
}

// readPragmas are what a connection that only reads opens with. The journal
// mode is not among them: it is persisted in the database header, and setting
// it is a write.
var readPragmas = []string{
	"query_only(1)",
	"busy_timeout(5000)",
	"cache_size(-65536)",
}

// OpenToRead opens the cache for asking, and never for building.
//
// A database that is not there is not made: the index is built by the
// application that scans, and a machine where that has never run has an index
// with nothing in it to read. Saying so is the answer; an empty database made
// here would say the vaults are empty instead.
func OpenToRead(ctx context.Context, path string) (*Reading, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	read, err := sql.Open("sqlite", dsnOf(path, readPragmas))
	if err != nil {
		return nil, err
	}
	if err := read.PingContext(ctx); err != nil {
		read.Close()
		return nil, err
	}
	return &Reading{read: read}, nil
}

// OpenNothing is an index holding nothing, for a machine where nothing has
// scanned yet.
//
// It is made in memory and goes with the process. One connection serves it,
// because a second would open a database of its own and an empty index would
// then differ from itself between two questions.
func OpenNothing(ctx context.Context) (*Reading, error) {
	db, err := sql.Open("sqlite", dsn(":memory:"))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := migrate(ctx, db); err != nil {
		db.Close()
		return nil, err
	}
	return &Reading{read: db}, nil
}

func (r *Reading) Close() error { return r.read.Close() }

// NoteQueries is what the notes are asked through, and the whole of what this
// opening offers.
func (r *Reading) NoteQueries() *note.Queries { return note.NewQueries(r.read) }

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

// Statistics writes, so it takes the pool that is allowed to.
func (d *DB) Statistics() Statistics { return Statistics{d.write} }

// NoteQueries reads, so it takes the pool that does not wait for the writer.
func (d *DB) NoteQueries() *note.Queries { return note.NewQueries(d.read) }

// ChunkQueries reads, so a search answers while a scan is still writing.
func (d *DB) ChunkQueries() *chunk.Queries { return chunk.NewQueries(d.read) }

// Sources is the source and vector ports over the chunk tables. It writes and
// reads both, through the pool each half belongs to.
func (d *DB) Sources() sources { return sources{write: d.Chunks(), read: d.ChunkQueries()} }

func dsn(path string) string { return dsnOf(path, pragmas) }

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
