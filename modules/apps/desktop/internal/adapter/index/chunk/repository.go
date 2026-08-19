// Package chunk stores the windows a source's text is cut into, and the vectors
// made from them.
//
// The text itself is not stored. A chunk is a place in a file — where it starts
// and how long it is — so showing a passage means reading the source again.
package chunk

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"

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
// `Text` is what the window holds, and is indexed and hashed, not kept: it is
// the text at `Start` for `Length` in the file, so a window whose text says
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
	Chunk       int64
	Fingerprint []byte
	Recipe      string
	Model       string
	Dims        int
	Kind        string
	Value       []byte
	Coarse      []byte
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
	if err := Replace(ctx, tx, source, vault, windows); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// SaveWindows makes the chunks of one source the windows given.
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
	if err := Replace(ctx, tx, source, vault, windows); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// SaveVectors writes both representations of each vector.
//
// They go in one transaction, and both are written by reading the chunk they
// belong to. A chunk with only one of the two is absent from the coarse pass
// and invisible to the question of what has no vector.
//
// A vector naming a chunk the index no longer holds is written nowhere, and
// what is left of the group is written as it stands.
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
		if err := exec(ctx, tx, "save_vector", v.Model, v.Dims, v.Kind, v.Value, v.Chunk); err != nil {
			return err
		}
		// Kept by the text it was made from, where renumbering cannot reach it.
		if len(v.Fingerprint) > 0 {
			if err := exec(ctx, tx, "keep_vector", v.Fingerprint, v.Recipe, v.Value); err != nil {
				return err
			}
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

// Replace makes the chunks of one source the windows given, inside a
// transaction that is already open.
//
// A chunk is identified by the hash of its text. A window whose hash is on a row
// of this source keeps that row, and its vector and its full-text row with it;
// the row is moved to where the text now is. A window whose hash is on no row is
// a new chunk, and a row whose hash is in no window is a chunk that is gone.
//
// A large window covers the whole of a note, so its hash moves whenever the note
// is edited at all, and `chunks.parent … ON DELETE CASCADE` takes every window
// inside a large one with it. The new rows go in first, the windows that were
// kept are then pointed at the large window they now sit in, and the rows that
// are gone come out last.
//
// Every window written is indexed for the words it holds, large and small alike,
// so that the lexical and the dense half of a search name one kind of row.
func Replace(ctx context.Context, tx *sql.Tx, source, vault int64, windows []Window) error {
	held, err := chunksOf(ctx, tx, source)
	if err != nil {
		return err
	}

	w, err := prepare(ctx, tx)
	if err != nil {
		return err
	}
	defer w.close()

	for _, large := range windows {
		row, err := w.put(ctx, held, source, vault, large, nil)
		if err != nil {
			return err
		}
		for _, small := range large.Small {
			if _, err := w.put(ctx, held, source, vault, small, row); err != nil {
				return err
			}
		}
	}
	if err := remove(ctx, tx, held.unclaimed()); err != nil {
		return err
	}
	return forget(ctx, tx, held.forgotten())
}

// writer is the three statements a cut runs per window, prepared once for the
// whole source.
type writer struct{ insert, index, move *sql.Stmt }

func prepare(ctx context.Context, tx *sql.Tx) (writer, error) {
	var w writer
	for _, s := range []struct {
		name string
		at   **sql.Stmt
	}{
		{"insert_chunk", &w.insert},
		{"insert_fts", &w.index},
		{"move_chunk", &w.move},
	} {
		prepared, err := tx.PrepareContext(ctx, stmt.Get(s.name))
		if err != nil {
			w.close()
			return writer{}, fmt.Errorf("%s: %w", s.name, err)
		}
		*s.at = prepared
	}
	return w, nil
}

func (w writer) close() {
	for _, s := range []*sql.Stmt{w.insert, w.index, w.move} {
		if s != nil {
			s.Close()
		}
	}
}

// put is the row one window is held on, and moves or writes it.
//
// A window inside another arrives with the row enclosing it, and a large window
// with nothing, which is also what the row's `parent` becomes.
func (w writer) put(ctx context.Context, held *held, source, vault int64, win Window, parent any) (int64, error) {
	key := text{hash: hashOf(win.Text), small: parent != nil}
	if row, kept := held.claim(key); kept {
		if _, err := w.move.ExecContext(ctx, win.Start, win.Length, parent, nullable(win.Location), row); err != nil {
			return 0, fmt.Errorf("move_chunk: %w", err)
		}
		return row, nil
	}

	var row int64
	err := w.insert.QueryRowContext(ctx,
		source, vault, win.Start, win.Length, parent, nullable(win.Location), key.hash).Scan(&row)
	if err != nil {
		return 0, fmt.Errorf("insert_chunk: %w", err)
	}
	if _, err := w.index.ExecContext(ctx, row, win.Text); err != nil {
		return 0, fmt.Errorf("insert_fts: %w", err)
	}
	return row, nil
}

// text is what a window has to hold to be held on a row: the same text, cut at
// the same size. A vector belongs to a window that sits inside another, so the
// two sizes are separate populations.
type text struct {
	hash  string
	small bool
}

// held is what a source's rows hold, in the shape a fresh cut asks about them.
type held struct {
	rows map[text][]int64
	left map[int64]bool
	// text is the fingerprint each row holds, so a row that goes says which
	// text went with it.
	text map[int64]string
}

func chunksOf(ctx context.Context, tx *sql.Tx, source int64) (*held, error) {
	rows, err := tx.QueryContext(ctx, stmt.Get("chunks_of"), source)
	if err != nil {
		return nil, fmt.Errorf("chunks_of: %w", err)
	}
	defer rows.Close()

	h := &held{rows: map[text][]int64{}, left: map[int64]bool{}, text: map[int64]string{}}
	for rows.Next() {
		var row int64
		var key text
		if err := rows.Scan(&row, &key.hash, &key.small); err != nil {
			return nil, fmt.Errorf("chunks_of: %w", err)
		}
		h.rows[key] = append(h.rows[key], row)
		h.left[row] = true
		h.text[row] = key.hash
	}
	return h, rows.Err()
}

// claim is a row holding the text given, and false where none does. A row is
// claimed once, so a text that occurs twice in a source is two rows.
func (h *held) claim(key text) (int64, bool) {
	rows := h.rows[key]
	if len(rows) == 0 {
		return 0, false
	}
	row := rows[0]
	h.rows[key] = rows[1:]
	delete(h.left, row)
	return row, true
}

// unclaimed is the rows of the source no window holds, in order, so that a cut
// writes the same thing twice running.
func (h *held) unclaimed() []int64 {
	out := make([]int64, 0, len(h.left))
	for row := range h.left {
		out = append(out, row)
	}
	slices.Sort(out)
	return out
}

// remove takes out the rows given, and everything indexed over them.
//
// The rows in the two virtual tables go first, by the chunk's own number.
// Nothing cascades into a virtual table, and a row left in either answers a
// search with a chunk that no longer exists. A large window takes the windows
// inside it, so a row here may already be gone from `chunks` by the time it is
// reached.
func remove(ctx context.Context, tx *sql.Tx, rows []int64) error {
	for _, row := range rows {
		for _, name := range []string{"delete_fts", "delete_vec", "delete_chunk"} {
			if err := exec(ctx, tx, name, row); err != nil {
				return err
			}
		}
	}
	return nil
}

// hashOf addresses a window by the text it holds. A chunk keeps its row, and its
// vector and its full-text row with it, for as long as this value stays the same.
func hashOf(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
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

// forgotten is the text of the rows no window holds: what this source used to
// hold and does not any more.
func (h *held) forgotten() []string {
	out := make([]string, 0, len(h.left))
	for row := range h.left {
		if hash := h.text[row]; hash != "" {
			out = append(out, hash)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// forget takes out the vectors made for text no chunk holds.
//
// It is asked where a source was cut again, so the text is known to have been
// replaced. A source the vault no longer offers takes nothing with it: a folder
// that could not be read looks the same from here as one whose files were
// deleted, and a vector was bought.
func forget(ctx context.Context, tx *sql.Tx, texts []string) error {
	for _, hash := range texts {
		if err := exec(ctx, tx, "forget_vector", hash, hash); err != nil {
			return err
		}
	}
	return nil
}
