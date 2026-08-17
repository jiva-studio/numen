// Package chunk stores the windows a source's text is cut into, and the vectors
// made from them.
//
// The text itself is not stored. A chunk is a place in a file — where it starts
// and how long it is — so showing a passage means reading the source again.
package chunk

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/index/sqlfile"
)

//go:embed sql/*.sql
var files embed.FS

var stmt = sqlfile.Load(files, "sql")

// Statements exposes the SQL this package runs, for a test that asks the
// database how it intends to answer each one.
func Statements() map[string]string { return stmt }

// errNoVault is returned when the index has never been told about the vault
// being asked about. Every question about a vault it does not know has the same
// answer — nothing — so callers turn this into an empty result.
var errNoVault = errors.New("vault not in the index")

// Source is a file the index has read, and what reading it produced. `Hash` and
// `Recipe` are empty until something computes them.
type Source struct {
	Path   string
	Kind   string
	Size   int64
	MTime  int64
	Hash   string
	Recipe string
}

// Window is one cut of a source's text. `Location` is what the source's own
// numbering calls the place, and is empty when the format offered none.
//
// `Text` is what the window holds, and is indexed, not kept: it is the
// text at `Start` for `Length` in the file, so a window whose text says
// something the file does not is a window that cannot be read back.
//
// `Small` are the windows inside this one. A Window with none of its own is a
// large window all the same: what makes it large is that nothing encloses it.
type Window struct {
	Start    int
	Length   int
	Location string
	Text     string
	Small    []Window
}

// Vector is one chunk's embedding in both representations that are stored.
//
// `Kind` is the quantisation of `Value` — `int8` or `float32` — and is recorded
// because a blob does not say what it holds.
type Vector struct {
	Chunk  int64
	Model  string
	Dims   int
	Kind   string
	Value  []byte
	Coarse []byte
}

// Repository is the collection of chunks and the vectors made from them.
type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

// SaveSource records one file of one kind and what reading it produced.
func (r *Repository) SaveSource(ctx context.Context, vaultID string, s Source) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback()

	vault, err := vaultRow(ctx, tx, vaultID)
	if err != nil {
		return err
	}
	var row int64
	if err := tx.QueryRowContext(ctx, stmt.Get("save_source"),
		vault, s.Path, s.Kind, s.Size, s.MTime, nullable(s.Hash), nullable(s.Recipe)).Scan(&row); err != nil {
		return fmt.Errorf("save_source %s: %w", s.Path, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// SaveExtraction records what a file is and replaces its chunks with the windows
// its text was cut into, in one transaction.
//
// One write, because a recipe names the sizes a source's chunks were cut into:
// the recipe and the chunks it describes are recorded together.
func (r *Repository) SaveExtraction(ctx context.Context, vaultID string, s Source, windows []Window) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback()

	vault, err := vaultRow(ctx, tx, vaultID)
	if err != nil {
		return err
	}
	var source int64
	if err := tx.QueryRowContext(ctx, stmt.Get("save_source"),
		vault, s.Path, s.Kind, s.Size, s.MTime, nullable(s.Hash), nullable(s.Recipe)).Scan(&source); err != nil {
		return fmt.Errorf("save_source %s: %w", s.Path, err)
	}
	if err := Clear(ctx, tx, source); err != nil {
		return err
	}
	if err := Write(ctx, tx, source, vault, windows); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// SaveWindows replaces the chunks of one source with the windows given.
func (r *Repository) SaveWindows(ctx context.Context, vaultID, kind, path string, windows []Window) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback()

	vault, err := vaultRow(ctx, tx, vaultID)
	if err != nil {
		return err
	}
	source, err := identify(ctx, tx, vault, kind, path)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%s is not a %s the index holds", path, kind)
	}
	if err != nil {
		return err
	}
	if err := Clear(ctx, tx, source); err != nil {
		return err
	}
	if err := Write(ctx, tx, source, vault, windows); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// SaveVectors writes both representations of each vector.
//
// They go in one transaction. A chunk with only one of the two is absent from
// the coarse pass and invisible to the question of what has no vector.
func (r *Repository) SaveVectors(ctx context.Context, vectors []Vector) error {
	if len(vectors) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback()

	for _, v := range vectors {
		// The vector index takes no conflict clause, so a row that is being
		// replaced is removed first.
		if err := exec(ctx, tx, "delete_vec", v.Chunk); err != nil {
			return err
		}
		if err := exec(ctx, tx, "insert_vec", v.Coarse, v.Chunk); err != nil {
			return err
		}
		if err := exec(ctx, tx, "save_vector", v.Chunk, v.Model, v.Dims, v.Kind, v.Value); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// RemoveSources takes out the sources of one kind whose files are gone. The
// vault is authoritative: what is not on disk is not in the index.
func (r *Repository) RemoveSources(ctx context.Context, vaultID, kind string, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback()

	vault, err := vaultRow(ctx, tx, vaultID)
	if err != nil {
		return err
	}
	for _, path := range paths {
		source, err := identify(ctx, tx, vault, kind, path)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		if err := Clear(ctx, tx, source); err != nil {
			return err
		}
		if err := exec(ctx, tx, "delete_source", source); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// Clear takes out the chunks of one source, and everything indexed over them.
//
// The rows in the two virtual tables go first, by the chunk's own number.
// Nothing cascades into a virtual table, and a row left in either answers a
// search with a chunk that no longer exists.
func Clear(ctx context.Context, tx *sql.Tx, source int64) error {
	for _, name := range []string{"clear_fts", "clear_vec"} {
		if err := exec(ctx, tx, name, source); err != nil {
			return err
		}
	}
	return exec(ctx, tx, "clear_chunks", source)
}

// Write puts the windows of one source in, inside a transaction that is already
// open. A large window is written before the windows inside it, which is what
// gives them something to point at.
//
// Every window written is indexed for the words it holds, large and small alike,
// so that the lexical and the dense half of a search name one kind of row.
func Write(ctx context.Context, tx *sql.Tx, source, vault int64, windows []Window) error {
	insert, err := tx.PrepareContext(ctx, stmt.Get("insert_chunk"))
	if err != nil {
		return fmt.Errorf("insert_chunk: %w", err)
	}
	defer insert.Close()

	index, err := tx.PrepareContext(ctx, stmt.Get("insert_fts"))
	if err != nil {
		return fmt.Errorf("insert_fts: %w", err)
	}
	defer index.Close()

	for _, large := range windows {
		row, err := write(ctx, insert, index, source, vault, large, nil)
		if err != nil {
			return err
		}
		for _, small := range large.Small {
			if _, err := write(ctx, insert, index, source, vault, small, row); err != nil {
				return err
			}
		}
	}
	return nil
}

func write(ctx context.Context, insert, index *sql.Stmt, source, vault int64, w Window, parent any) (int64, error) {
	var row int64
	err := insert.QueryRowContext(ctx, source, vault, w.Start, w.Length, parent, nullable(w.Location)).Scan(&row)
	if err != nil {
		return 0, fmt.Errorf("insert_chunk: %w", err)
	}
	if _, err := index.ExecContext(ctx, row, w.Text); err != nil {
		return 0, fmt.Errorf("insert_fts: %w", err)
	}
	return row, nil
}

// exec runs a named statement and says which one failed.
func exec(ctx context.Context, tx *sql.Tx, name string, args ...any) error {
	if _, err := tx.ExecContext(ctx, stmt.Get(name), args...); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// querier is anything that can answer a single-row question, so that this works
// inside a transaction as well as outside one.
type querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// vaultRow turns the identifier a vault carries in the world into the row
// number this database files it under.
func vaultRow(ctx context.Context, db querier, identifier string) (int64, error) {
	var row int64
	err := db.QueryRowContext(ctx, stmt.Get("vault_row"), identifier).Scan(&row)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, errNoVault
	}
	return row, err
}

// identify is the row number of one source of one kind.
func identify(ctx context.Context, db querier, vault int64, kind, path string) (int64, error) {
	var row int64
	err := db.QueryRowContext(ctx, stmt.Get("identify"), vault, kind, path).Scan(&row)
	return row, err
}

// nullable keeps an empty string out of the database, so that "nothing was
// written" and "an empty value was written" stay different questions.
func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
