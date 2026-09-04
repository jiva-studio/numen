// Package chunk stores the chunks a source's text is cut into, and the vectors
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
	"strings"
	"unicode/utf8"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/sqlfile"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
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

// Source is a file the index has read, and what reading it produced. `Hash`,
// `Recipe` and `TextFrom` are empty until something computes them.
type Source struct {
	Path   string
	Kind   string
	Size   int64
	MTime  int64
	Hash   string
	Recipe string

	// TextFrom names the producer of the text this source's chunks are places
	// in. Empty where the source's own bytes are the text.
	TextFrom string
}

// Chunk is one cut of a source's text. `Location` is where it sits in the terms
// the source's own numbering uses, and is empty when the format offered none.
//
// `Text` is what the chunk holds, and is indexed and hashed, not kept: it is
// the text at `Start` for `Length` in the file, so a chunk whose text says
// something the file does not is a chunk that cannot be read back.
//
// `Small` are the chunks inside this one. A Chunk with none of its own is a
// large chunk all the same: what makes it large is that nothing encloses it.
type Chunk struct {
	Start    int
	Length   int
	Location string
	Text     string
	// Opens are the parts of the source that begin exactly where this chunk
	// does: what a section starting here is called. Empty for a chunk that
	// opens none, which is most of them.
	Opens []string
	Small []Chunk
}

// Vector is one chunk's embedding in both representations that are stored.
//
// `Value` is one byte a dimension, for the rerank. `Coarse` is one bit a
// dimension, read out of `Value`, and is what the first pass compares. `Hash`
// addresses the text the vector was bought for, and `Recipe` names the model
// and the shape it was bought under.
type Vector struct {
	Chunk  int64
	Hash   []byte
	Recipe string
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
		vault, s.Path, s.Kind, s.Size, s.MTime, nullable(s.Hash), nullable(s.Recipe), nullable(s.TextFrom)).Scan(&row); err != nil {
		return fmt.Errorf("save_source %s: %w", s.Path, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// SaveExtraction records what a file is and replaces its chunks with the ones
// its text was cut into, in one transaction.
//
// One write, because a recipe names the sizes a source's chunks were cut into:
// the recipe and the chunks it describes are recorded together.
func (r *Repository) SaveExtraction(ctx context.Context, vaultID string, s Source, chunks []Chunk) error {
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
		vault, s.Path, s.Kind, s.Size, s.MTime, nullable(s.Hash), nullable(s.Recipe), nullable(s.TextFrom)).Scan(&source); err != nil {
		return fmt.Errorf("save_source %s: %w", s.Path, err)
	}
	if err := Replace(ctx, tx, source, vault, chunks); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// SaveChunks makes the chunks of one source the ones given.
func (r *Repository) SaveChunks(ctx context.Context, vaultID, kind, path string, chunks []Chunk) error {
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
	if err := Replace(ctx, tx, source, vault, chunks); err != nil {
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
		// The coarse index takes no conflict clause, so a row that is being
		// replaced is removed first.
		if err := exec(ctx, tx, "delete_vec", v.Chunk); err != nil {
			return err
		}
		if err := exec(ctx, tx, "insert_vec", v.Coarse, v.Chunk); err != nil {
			return err
		}
		if err := exec(ctx, tx, "keep_vector", v.Hash, v.Recipe, v.Value); err != nil {
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

// MoveSources files what the vault held at one path, and everything under it,
// where it now is. The chunks, the vectors, the links and the headings hang off
// the source's own number and travel with it untouched.
//
// Only the file at the path itself can be called something else afterwards;
// everything under a folder keeps the name it has.
func (r *Repository) MoveSources(ctx context.Context, vaultID, from, to string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback()

	vault, err := vaultRow(ctx, tx, vaultID)
	if errors.Is(err, errNoVault) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := displace(ctx, tx, vault, from, to); err != nil {
		return err
	}
	if err := rename(ctx, tx, vault, from, to); err != nil {
		return err
	}
	first, past := under(from)
	// What a path keeps is counted off it in characters: that is what the
	// statement cuts by, and a path is not Latin alone.
	kept := utf8.RuneCountInString(from) + 1
	if err := exec(ctx, tx, "move_sources", to, kept, vault, from, first, past); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// displace takes out what the vault holds where a move is about to land.
//
// A file is moved on disk and then moved here, and a scan reading the vault
// between the two files it afresh under its new path. The row that travels
// carries the chunks, the vectors and the links of the note, so it is the one
// that keeps the path, and the row standing there comes out.
//
// The rows that are moving stand still: a folder moved inside itself holds
// them, and they are the ones about to be filed.
func displace(ctx context.Context, tx *sql.Tx, vault int64, from, to string) error {
	first, past := under(to)
	rows, err := tx.QueryContext(ctx, stmt.Get("sources_at"), vault, to, first, past)
	if err != nil {
		return fmt.Errorf("sources_at: %w", err)
	}
	defer rows.Close()

	movingFirst, movingPast := under(from)
	var standing []int64
	for rows.Next() {
		var source int64
		var path string
		if err := rows.Scan(&source, &path); err != nil {
			return fmt.Errorf("sources_at: %w", err)
		}
		if path == from || (path >= movingFirst && path < movingPast) {
			continue
		}
		standing = append(standing, source)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("sources_at: %w", err)
	}
	rows.Close()

	for _, source := range standing {
		if err := Clear(ctx, tx, source); err != nil {
			return err
		}
		if err := exec(ctx, tx, "delete_source", source); err != nil {
			return err
		}
	}
	return nil
}

// rename says what the note at a path is called once its file is at another.
//
// A note is called by its `title` key, else by its filename. It takes the name
// of the file it lands under where the file carries no key, and where that
// filename is the one its own title is filed under; otherwise it carries the
// name it has.
//
// The name goes into the title index with it: that is what a search by name
// ranks and highlights against.
func rename(ctx context.Context, tx *sql.Tx, vault int64, from, to string) error {
	name := domain.Basename(to)
	if name == domain.Basename(from) {
		return nil
	}

	var title string
	var named bool
	err := tx.QueryRowContext(ctx, stmt.Get("note_naming"), vault, from).Scan(&title, &named)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("note_naming %s: %w", from, err)
	}
	shown := title
	if !named {
		shown = name
	} else if filed, _ := domain.Filename(title); domain.FoldName(filed) != domain.FoldName(name) {
		return nil
	}
	if err := exec(ctx, tx, "rename_note", domain.FoldName(name), shown, vault, from); err != nil {
		return err
	}
	return exec(ctx, tx, "rename_title", vault, from, shown)
}

// under is the range every path a folder holds falls in: from the folder's
// slash to the byte after one.
func under(folder string) (first, past string) {
	return folder + "/", folder + "0"
}

// Clear takes out the chunks of one source, and everything indexed over them.
//
// The rows in the three virtual tables go first, by the chunk's own number.
// Nothing cascades into a virtual table, and a chunk's number is handed to the
// next chunk that wants one, so a row left behind answers for that one.
func Clear(ctx context.Context, tx *sql.Tx, source int64) error {
	for _, name := range []string{"clear_fts", "clear_vec", "clear_parts"} {
		if err := exec(ctx, tx, name, source); err != nil {
			return err
		}
	}
	return exec(ctx, tx, "clear_chunks", source)
}

// Replace makes the chunks of one source the ones given, inside a transaction
// that is already open.
//
// A chunk is identified by the hash of its text. A chunk whose hash is on a row
// of this source keeps that row, and its vector and its full-text row with it;
// the row is moved to where the text now is. A chunk whose hash is on no row is
// a new chunk, and a row whose hash is in no chunk is a chunk that is gone.
//
// A large chunk covers the whole of a note, so its hash moves whenever the note
// is edited at all, and `chunks.parent … ON DELETE CASCADE` takes every chunk
// inside a large one with it. The new rows go in first, the chunks that were
// kept are then pointed at the large chunk they now sit in, and the rows that
// are gone come out last.
//
// Every chunk written is indexed for the words it holds, large and small alike,
// so that a search asked by words and one asked by meaning name one kind of row.
func Replace(ctx context.Context, tx *sql.Tx, source, vault int64, chunks []Chunk) error {
	held, err := chunksOf(ctx, tx, source)
	if err != nil {
		return err
	}
	// A chunk that says the same thing keeps its row through a cut, so the
	// names of the parts are dropped by the source and not with the chunks.
	if _, err := tx.ExecContext(ctx, stmt.Get("clear_parts"), source); err != nil {
		return fmt.Errorf("clear_parts: %w", err)
	}

	w, err := prepare(ctx, tx)
	if err != nil {
		return err
	}
	defer w.close()

	for _, large := range chunks {
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

// statements are what a cut runs per chunk, prepared once for the whole
// source.
type statements struct{ insert, index, names, move *sql.Stmt }

func prepare(ctx context.Context, tx *sql.Tx) (statements, error) {
	var w statements
	for _, s := range []struct {
		name string
		at   **sql.Stmt
	}{
		{"insert_chunk", &w.insert},
		{"insert_fts", &w.index},
		{"insert_part", &w.names},
		{"move_chunk", &w.move},
	} {
		prepared, err := tx.PrepareContext(ctx, stmt.Get(s.name))
		if err != nil {
			w.close()
			return statements{}, fmt.Errorf("%s: %w", s.name, err)
		}
		*s.at = prepared
	}
	return w, nil
}

func (w statements) close() {
	for _, s := range []*sql.Stmt{w.insert, w.index, w.names, w.move} {
		if s != nil {
			s.Close()
		}
	}
}

// put is the row one chunk is held on, and moves or writes it.
//
// A chunk inside another arrives with the row enclosing it, and a large chunk
// with nothing, which is also what the row's `parent` becomes.
func (w statements) put(ctx context.Context, held *rows, source, vault int64, c Chunk, parent any) (int64, error) {
	key := textID{hash: hashOf(c.Text), small: parent != nil}
	if row, kept := held.claim(key); kept {
		if _, err := w.move.ExecContext(ctx, c.Start, c.Length, parent, nullable(c.Location), row); err != nil {
			return 0, fmt.Errorf("move_chunk: %w", err)
		}
		if err := w.opens(ctx, row, c); err != nil {
			return 0, err
		}
		return row, nil
	}

	var row int64
	err := w.insert.QueryRowContext(ctx,
		source, vault, c.Start, c.Length, parent, nullable(c.Location), key.hash).Scan(&row)
	if err != nil {
		return 0, fmt.Errorf("insert_chunk: %w", err)
	}
	if _, err := w.index.ExecContext(ctx, row, c.Text); err != nil {
		return 0, fmt.Errorf("insert_fts: %w", err)
	}
	if err := w.opens(ctx, row, c); err != nil {
		return 0, err
	}
	return row, nil
}

// opens keeps the names of the parts one chunk begins, so a section can be
// found by its name and answer with the chunk it opens.
func (w statements) opens(ctx context.Context, row int64, c Chunk) error {
	if len(c.Opens) == 0 {
		return nil
	}
	if _, err := w.names.ExecContext(ctx, row, strings.Join(c.Opens, "\n")); err != nil {
		return fmt.Errorf("insert_part: %w", err)
	}
	return nil
}

// textID is what a chunk has to hold to be held on a row: the same text, cut at
// the same size. A vector belongs to a chunk that sits inside another, so the
// two sizes are separate populations.
type textID struct {
	hash  string
	small bool
}

// rows is what a source's rows hold, in the shape a fresh cut asks about them.
type rows struct {
	candidates map[textID][]int64
	left       map[int64]bool
	// hash is the fingerprint each row holds, so a row that goes says which
	// text went with it.
	hash map[int64]string
}

func chunksOf(ctx context.Context, tx *sql.Tx, source int64) (*rows, error) {
	cursor, err := tx.QueryContext(ctx, stmt.Get("chunks_of"), source)
	if err != nil {
		return nil, fmt.Errorf("chunks_of: %w", err)
	}
	defer cursor.Close()

	h := &rows{candidates: map[textID][]int64{}, left: map[int64]bool{}, hash: map[int64]string{}}
	for cursor.Next() {
		var row int64
		var key textID
		if err := cursor.Scan(&row, &key.hash, &key.small); err != nil {
			return nil, fmt.Errorf("chunks_of: %w", err)
		}
		h.candidates[key] = append(h.candidates[key], row)
		h.left[row] = true
		h.hash[row] = key.hash
	}
	return h, cursor.Err()
}

// claim is a row holding the text given, and false where none does. A row is
// claimed once, so a text that occurs twice in a source is two rows.
func (h *rows) claim(key textID) (int64, bool) {
	rows := h.candidates[key]
	if len(rows) == 0 {
		return 0, false
	}
	row := rows[0]
	h.candidates[key] = rows[1:]
	delete(h.left, row)
	return row, true
}

// unclaimed is the rows of the source no chunk holds, in order, so that a cut
// writes the same thing twice running.
func (h *rows) unclaimed() []int64 {
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
// search with a chunk that no longer exists. A large chunk takes the chunks
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

// hashOf addresses a chunk by the text it holds. A chunk keeps its row, and its
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

// forgotten is the text of the rows no chunk holds: what this source used to
// hold and does not any more.
func (h *rows) forgotten() []string {
	out := make([]string, 0, len(h.left))
	for row := range h.left {
		if hash := h.hash[row]; hash != "" {
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
