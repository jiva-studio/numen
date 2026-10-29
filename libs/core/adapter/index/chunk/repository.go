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
	"time"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/core/adapter/index/sqlfile"
	"github.com/jiva-studio/numen/libs/core/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

//go:embed sql/*.sql
var files embed.FS

var stmt = sqlfile.Load(files, "sql")

// Statements exposes the SQL this package runs, for tests that inspect the
// query plan of each statement.
func Statements() map[string]string { return stmt }

// errNoVault is returned when the index has no record of the requested vault.
// Every query for an unknown vault has the same result, an empty one, so
// callers convert this error into an empty result.
var errNoVault = errors.New("vault not in the index")

// Source is a file the index has read, and what reading it produced. `Hash`,
// `Recipe` and `Producer` are empty until something computes them.
type Source struct {
	Path   string
	Kind   string
	Size   int64
	MTime  int64
	Hash   string
	Recipe string

	// Producer is what made the text this source's chunks are places in. Empty
	// where the source's own bytes are the text.
	Producer string
}

// ConvertNanosToTime and ConvertTimeToNanos are the source row's modification
// time either way round.
// The column is an INTEGER of nanoseconds since the epoch, and zero is a row
// that names no time.

func ConvertNanosToTime(nanos int64) time.Time {
	if nanos == 0 {
		return time.Time{}
	}
	return time.Unix(0, nanos)
}

func ConvertTimeToNanos(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixNano()
}

// Chunk is one segment of a source's text. `Location` is the position in the
// source's own numbering and is empty when the format provides none.
//
// `Text` is the chunk content. It is indexed and hashed but not stored: it is
// the text at `Start` for `Length` in the file, so a chunk whose text differs
// from the file cannot be read back.
//
// `Small` are the chunks inside this one. A Chunk with none of its own is
// still a large chunk: it is large because nothing encloses it.
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

// FromDomain is the chunks of a source as the index holds them.
func FromDomain(in []domain.Chunk) []Chunk {
	out := make([]Chunk, 0, len(in))
	for _, c := range in {
		out = append(out, Chunk{
			Start:    c.Start,
			Length:   c.Length,
			Location: c.Location,
			Text:     c.Text,
			Opens:    c.Opens,
			Small:    FromDomain(c.Small),
		})
	}
	return out
}

// Repository is the collection of chunks and the vectors made from them.
type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

// SaveSource records one file of one kind and what reading it produced.
func (r *Repository) SaveSource(ctx context.Context, vaultID domain.VaultID, s Source) error {
	tx, err := sqlwrite.Begin(ctx, r.db)
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
		vault, s.Path, s.Kind, s.Size, s.MTime, newNullable(s.Hash), newNullable(s.Recipe), newNullable(s.Producer)).Scan(&row); err != nil {
		return fmt.Errorf("record the source %s: %w", s.Path, err)
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
func (r *Repository) SaveExtraction(ctx context.Context, vaultID domain.VaultID, s Source, chunks []Chunk) error {
	tx, err := sqlwrite.Begin(ctx, r.db)
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
		vault, s.Path, s.Kind, s.Size, s.MTime, newNullable(s.Hash), newNullable(s.Recipe), newNullable(s.Producer)).Scan(&source); err != nil {
		return fmt.Errorf("record the source %s: %w", s.Path, err)
	}
	if err := Replace(ctx, tx, source, vault, chunks); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// ReplaceChunks looks up the source this vault holds at a path and replaces
// its chunks, in a transaction of its own. The source is already recorded or
// there is nothing to replace the chunks of.
func (r *Repository) ReplaceChunks(ctx context.Context, vaultID domain.VaultID, kind, path string, chunks []Chunk) error {
	tx, err := sqlwrite.Begin(ctx, r.db)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback()

	vault, err := vaultRow(ctx, tx, vaultID)
	if err != nil {
		return err
	}
	source, err := getSourceRow(ctx, tx, vault, kind, path)
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

// RemoveSources takes out the sources of one kind whose files are gone. The
// vault is authoritative: what is not on disk is not in the index.
func (r *Repository) RemoveSources(ctx context.Context, vaultID domain.VaultID, kind string, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	tx, err := sqlwrite.Begin(ctx, r.db)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback()

	vault, err := vaultRow(ctx, tx, vaultID)
	if err != nil {
		return err
	}
	for _, path := range paths {
		source, err := getSourceRow(ctx, tx, vault, kind, path)
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
func (r *Repository) MoveSources(ctx context.Context, vaultID domain.VaultID, from, to string) error {
	tx, err := sqlwrite.Begin(ctx, r.db)
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
	if err := deleteMoveTarget(ctx, tx, vault, from, to); err != nil {
		return err
	}
	if err := updateNoteName(ctx, tx, vault, from, to); err != nil {
		return err
	}
	first, past := getRangeUnder(from)
	// What a path keeps is counted off it in characters: that is what the
	// statement cuts by, and a path is not Latin alone.
	fromLen := utf8.RuneCountInString(from) + 1
	if err := exec(ctx, tx, "move_sources", to, fromLen, vault, from, first, past); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// deleteMoveTarget deletes the rows the vault holds at the path a move lands on.
//
// A file is moved on disk and then in the index. A scan between the two steps
// indexes the file again under its new path. The moved row owns the note's
// chunks, vectors and links, so it keeps the path and the row already at that
// path is deleted.
//
// Rows that are being moved are kept: a folder moved inside itself contains
// them, and they are the rows to be renamed.
func deleteMoveTarget(ctx context.Context, tx *sqlwrite.Transaction, vault int64, from, to string) error {
	first, past := getRangeUnder(to)
	rows, err := tx.QueryContext(ctx, stmt.Get("sources_under"), vault, to, first, past)
	if err != nil {
		return fmt.Errorf("what the vault holds at %s and under it: %w", to, err)
	}
	defer rows.Close()

	movingFirst, movingPast := getRangeUnder(from)
	var displaced []int64
	for rows.Next() {
		var source int64
		var path string
		if err := rows.Scan(&source, &path); err != nil {
			return fmt.Errorf("what the vault holds at %s and under it: %w", to, err)
		}
		if path == from || (path >= movingFirst && path < movingPast) {
			continue
		}
		displaced = append(displaced, source)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("what the vault holds at %s and under it: %w", to, err)
	}
	rows.Close()

	for _, source := range displaced {
		if err := Clear(ctx, tx, source); err != nil {
			return err
		}
		if err := exec(ctx, tx, "delete_source", source); err != nil {
			return err
		}
	}
	return nil
}

// updateNoteName updates the name of the note at a path after its file moved.
//
// A note is named by its `title` key, else by its filename. The name follows the
// new filename when the file has no `title` key and the old filename matches the
// note's name. In every other case the name is unchanged.
//
// The name is also written to the title index, which a search by name ranks and
// highlights against.
func updateNoteName(ctx context.Context, tx *sqlwrite.Transaction, vault int64, from, to string) error {
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
		return fmt.Errorf("what the note %s is called: %w", from, err)
	}
	displayName := title
	if !named {
		displayName = name
	} else if filed, _ := domain.GetReducedFilename(title); domain.FoldName(filed) != domain.FoldName(name) {
		return nil
	}
	if err := exec(ctx, tx, "rename_note", domain.FoldName(name), displayName, vault, from); err != nil {
		return err
	}
	return exec(ctx, tx, "rename_title", vault, from, displayName)
}

// getRangeUnder is the range every path a folder holds falls in: from the
// folder's slash to the byte after one.
func getRangeUnder(folder string) (first, past string) {
	return folder + "/", folder + "0"
}

// Clear takes out the chunks of one source, and everything indexed over them.
//
// The rows in the three virtual tables go first, by the chunk's own number.
// Nothing cascades into a virtual table, and a chunk's number is handed to the
func Clear(ctx context.Context, tx *sqlwrite.Transaction, source int64) error {
	for _, name := range []string{"clear_fts", "clear_vec", "clear_sections"} {
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
// is edited at all, and `chunks.parent_id … ON DELETE CASCADE` takes every
// chunk inside a large one with it. The new rows go in first, the chunks that
// were kept are then pointed at the large chunk they now sit in, and the rows
// that are gone come out last.
//
// Only leaf chunks are indexed for their words, so a word search finds the
// smallest enclosing passage.
func Replace(ctx context.Context, tx *sqlwrite.Transaction, source, vault int64, chunks []Chunk) error {
	target, err := chunksOf(ctx, tx, source)
	if err != nil {
		return err
	}
	// A chunk with unchanged text keeps its row, so section names are cleared per
	// source and not with the chunks.
	if _, err := tx.ExecContext(ctx, stmt.Get("clear_sections"), source); err != nil {
		return fmt.Errorf("take the names of this source's sections out of the index: %w", err)
	}

	for _, large := range chunks {
		row, err := upsertChunk(ctx, tx, target, source, vault, large, nil, len(large.Small) == 0)
		if err != nil {
			return err
		}
		for _, small := range large.Small {
			if _, err := upsertChunk(ctx, tx, target, source, vault, small, row, true); err != nil {
				return err
			}
		}
	}
	if err := deleteRows(ctx, tx, target.getUnusedRows()); err != nil {
		return err
	}
	return deleteOrphanVectors(ctx, tx, target.getRemovedHashes())
}

// upsertChunk is the row one chunk is held on, and moves or writes it.
//
// A chunk inside another arrives with the row enclosing it, and a large chunk
// with nothing, which is also what the row's `parent_id` becomes.
func upsertChunk(
	ctx context.Context, tx *sqlwrite.Transaction, target *rows,
	source, vault int64, c Chunk, parent any, isLeaf bool,
) (int64, error) {
	key := textID{hash: computeHash(c.Text), isSmall: parent != nil}
	if row, hasRow := target.takeUnusedRow(key); hasRow {
		if _, err := tx.ExecContext(ctx, stmt.Get("move_chunk"),
			c.Start, c.Length, parent, newNullable(c.Location), row); err != nil {
			return 0, fmt.Errorf("move a chunk to where its text now is: %w", err)
		}
		if err := writeSectionNames(ctx, tx, row, c); err != nil {
			return 0, err
		}
		return row, nil
	}

	var row int64
	err := tx.QueryRowContext(ctx, stmt.Get("insert_chunk"),
		source, vault, c.Start, c.Length, parent, newNullable(c.Location), key.hash).Scan(&row)
	if err != nil {
		return 0, fmt.Errorf("store a chunk of this source: %w", err)
	}
	if isLeaf {
		if _, err := tx.ExecContext(ctx, stmt.Get("insert_fts"), row, c.Text); err != nil {
			return 0, fmt.Errorf("index a chunk for the words in it: %w", err)
		}
	}
	if err := writeSectionNames(ctx, tx, row, c); err != nil {
		return 0, err
	}
	return row, nil
}

// writeSectionNames keeps the names of the sections one chunk begins, so a section can be
// found by its name and answer with the chunk it opens.
func writeSectionNames(ctx context.Context, tx *sqlwrite.Transaction, row int64, c Chunk) error {
	if len(c.Opens) == 0 {
		return nil
	}
	if _, err := tx.ExecContext(ctx, stmt.Get("insert_section"), row, strings.Join(c.Opens, "\n")); err != nil {
		return fmt.Errorf("index the names of the sections a chunk opens: %w", err)
	}
	return nil
}

// textID is what a chunk has to hold to be held on a row: the same text, cut at
// two sizes are separate populations.
type textID struct {
	hash    string
	isSmall bool
}

// rows holds the existing rows of a source, indexed for matching against new chunks.
type rows struct {
	candidates    map[textID][]int64
	remainingErrs map[int64]bool
	// hash maps each row to its text hash, which identifies the text of a
	// removed row.
	hash map[int64]string
}

func chunksOf(ctx context.Context, tx *sqlwrite.Transaction, source int64) (*rows, error) {
	cursor, err := tx.QueryContext(ctx, stmt.Get("chunks_of"), source)
	if err != nil {
		return nil, fmt.Errorf("the chunks this source is already cut into: %w", err)
	}
	defer cursor.Close()

	h := &rows{candidates: map[textID][]int64{}, remainingErrs: map[int64]bool{}, hash: map[int64]string{}}
	for cursor.Next() {
		var row int64
		var key textID
		if err := cursor.Scan(&row, &key.hash, &key.isSmall); err != nil {
			return nil, fmt.Errorf("the chunks this source is already cut into: %w", err)
		}
		h.candidates[key] = append(h.candidates[key], row)
		h.remainingErrs[row] = true
		h.hash[row] = key.hash
	}
	return h, cursor.Err()
}

// takeUnusedRow is a row holding the text given, and false where none does. A row is
// claimed once, so a text that occurs twice in a source is two rows.
func (h *rows) takeUnusedRow(key textID) (int64, bool) {
	rows := h.candidates[key]
	if len(rows) == 0 {
		return 0, false
	}
	row := rows[0]
	h.candidates[key] = rows[1:]
	delete(h.remainingErrs, row)
	return row, true
}

// getUnusedRows returns the rows of the source that no chunk matched, in order,
// so repeated runs write the same result.
func (h *rows) getUnusedRows() []int64 {
	out := make([]int64, 0, len(h.remainingErrs))
	for row := range h.remainingErrs {
		out = append(out, row)
	}
	slices.Sort(out)
	return out
}

// deleteRows deletes the given rows and everything indexed over them.
//
// The rows in the two virtual tables are deleted first, by chunk number.
// Nothing cascades into a virtual table, and a remaining row returns a search
// hit for a chunk that is deleted. A large chunk deletes the chunks inside it,
// so a row here may already be gone from `chunks` when it is processed.
func deleteRows(ctx context.Context, tx *sqlwrite.Transaction, rows []int64) error {
	for _, row := range rows {
		for _, name := range []string{"delete_fts", "delete_vec", "delete_chunk"} {
			if err := exec(ctx, tx, name, row); err != nil {
				return err
			}
		}
	}
	return nil
}

// computeHash addresses a chunk by the text it holds. A chunk keeps its row, and its
// vector and its full-text row with it, for as long as this value stays the same.
func computeHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// exec runs a named statement and wraps an error with the statement name.
func exec(ctx context.Context, tx *sqlwrite.Transaction, name string, args ...any) error {
	if _, err := tx.ExecContext(ctx, stmt.Get(name), args...); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// querier runs a single-row query, on a transaction or on a plain connection.
type querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// vaultRow turns the identifier a vault carries in the world into the row
// number this database files it under.
func vaultRow(ctx context.Context, db querier, identifier domain.VaultID) (int64, error) {
	var row int64
	err := db.QueryRowContext(ctx, stmt.Get("vault_row"), string(identifier)).Scan(&row)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, errNoVault
	}
	return row, err
}

// getSourceRow is the row number of one source of one kind.
func getSourceRow(ctx context.Context, db querier, vault int64, kind, path string) (int64, error) {
	var row int64
	err := db.QueryRowContext(ctx, stmt.Get("identify"), vault, kind, path).Scan(&row)
	return row, err
}

// newNullable maps an empty string to NULL, so an unset value and an empty
// value stay distinguishable.
func newNullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// getRemovedHashes returns the text hashes of the rows no chunk matched: the
// text the source contained before and does not contain now.
func (h *rows) getRemovedHashes() []string {
	out := make([]string, 0, len(h.remainingErrs))
	for row := range h.remainingErrs {
		if hash := h.hash[row]; hash != "" {
			out = append(out, hash)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}
