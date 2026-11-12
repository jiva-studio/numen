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
	"errors"
	"fmt"
	"net/url"
	"runtime"
	"slices"
	"strings"

	// The driver and the vector extension register themselves under the names
	// a DSN opens them by, and nothing here names either package.
	_ "modernc.org/sqlite"
	_ "modernc.org/sqlite/vec"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/chunk"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/note"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/opentabs"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/searchpolicy"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/version"
)

// DB owns the connections.
// There are two pools, because SQLite has one writer and any number of readers.
// The write pool is one connection, so writers queue in Go. The read pool is
// sized to the cores: in WAL mode a reader never waits for the writer.
// Several processes may open the one file; see sqlwrite for the retry.
type DB struct {
	write   *sql.DB
	read    *sql.DB
	minimum chunk.Minimum
}

func getPragmas() []string {
	return []string{
		// WAL so a long scan does not block readers. This one is persisted in the
		// database header, and belongs with the rest all the same.
		"journal_mode(WAL)",
		// Foreign keys so removing a vault cannot leave rows pointing at nothing.
		"foreign_keys(1)",
		// Wait for a writer, up to five seconds, before SQLITE_BUSY. The writer being
		// waited for may be in another process, and a write that exhausted the wait
		// retries.
		"busy_timeout(5000)",
		// Pages are read from the mapped file, shared by every connection, up to
		// 256 MiB of it.
		"mmap_size(268435456)",
		// Intermediate sort B-trees and temporary tables stay in memory.
		"temp_store(MEMORY)",
		"cache_size(-65536)",
	}
}

const (
	shipped        = "synchronous(NORMAL)"
	unsynchronized = "synchronous(OFF)"
)

// Option is what an index is opened with.
type Option func(*settings)

// settings are what one index is opened with, and nothing outside this call
// reads them.
type settings struct{ synchronous string }

// Unsynchronized opens an index that does not wait for the disk on a commit. WAL, the single
// writer and the busy timeout stay as they are; only the moment of the flush moves.
func Unsynchronized() Option {
	return func(s *settings) { s.synchronous = unsynchronized }
}

func Open(ctx context.Context, path string, opts ...Option) (*DB, error) {
	// A commit waits for the disk. The index is a cache: a crash costs a
	// rescan, never data, and NORMAL under WAL is what that is worth.
	pos := settings{synchronous: shipped}
	for _, one := range opts {
		one(&pos)
	}

	// A file this build cannot use is read before the pool below sets WAL on it.
	if err := checkFile(ctx, path); errors.Is(err, errUnknownVersion) {
		if err := removeIndexFiles(path); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	// The first connection reports an unusable path.
	write, err := sql.Open("sqlite", writeDSN(path, pos))
	if err != nil {
		return nil, fmt.Errorf("open the index at %s: %w", path, err)
	}
	// One writer. More connections would only mean more of them failing on a
	// busy lock.
	write.SetMaxOpenConns(1)

	if err := migrate(ctx, write, path); err != nil {
		return nil, errors.Join(err, write.Close())
	}

	read, err := sql.Open("sqlite", dsn(path, pos))
	if err != nil {
		return nil, errors.Join(fmt.Errorf("open the index at %s: %w", path, err), write.Close())
	}
	read.SetMaxOpenConns(max(4, runtime.NumCPU()))
	read.SetMaxIdleConns(max(2, runtime.NumCPU()/2))
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

// NoteProblems reads what the index found wrong in the notes.
func (d *DB) NoteProblems() *note.Problems { return note.NewProblems(d.read) }

// ChunkFiles reads, so a scan decides what to read again while a search answers.
func (d *DB) ChunkFiles() *chunk.Files { return chunk.NewFiles(d.read) }

// ChunkSearch reads, so a search answers while a scan is still writing.
func (d *DB) ChunkSearch() *chunk.Search { return chunk.NewSearch(d.read) }

// ChunkVectors reads the vectors and the chunks that lack one.
func (d *DB) ChunkVectors() *chunk.Vectors { return chunk.NewVectors(d.read, &d.minimum) }

// Sources is the source and vector ports over the chunk tables. It writes and
// reads both, through the pool each half belongs to.
func (d *DB) Sources() sources {
	return sources{queries: queries{files: d.ChunkFiles(), vectors: d.ChunkVectors()}, write: d.Chunks(), versions: d.TextVersions()}
}

// TextVersions is the text versions of every vault. A rebuild of the index
// reads the vaults again and leaves them as they are.
func (d *DB) TextVersions() *version.Repository { return version.NewRepository(d.read, d.write) }

// OpenTabs is the tabs the window has open on every vault. A rebuild of the
// index leaves them as they are.
func (d *DB) OpenTabs() *opentabs.Repository { return opentabs.NewRepository(d.read, d.write) }

// SearchFloors is the effective similarity floor of every vault under every recipe. A rebuild of
// the index leaves it as it is.
func (d *DB) SearchFloors() *searchpolicy.Repository {
	return searchpolicy.NewRepository(d.read, d.write)
}

// SearchSamples reads the titles and vectors a plugin measures the floor with.
func (d *DB) SearchSamples() *searchpolicy.Samples { return searchpolicy.NewSamples(d.read) }

func dsn(path string, pos settings) string {
	return dsnOf(path, append(slices.Clone(getPragmas()), pos.synchronous))
}

func writeDSN(path string, pos settings) string { return dsn(path, pos) + "&_txlock=immediate" }

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
