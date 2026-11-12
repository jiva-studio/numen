// Package chunk stores the chunks a source's text is cut into, and the vectors
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
	"time"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlfile"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/note/domain"
)

//go:embed sql/*.sql
var files embed.FS

func getStatement(name string) string { return sqlfile.Read(files, "sql", name) }

// errNoVault is returned when the index has no record of the requested vault.
// Every query for an unknown vault has the same result, an empty one, so
// callers convert this error into an empty result.
var errNoVault = errors.New("vault not in the index")

// Source is a file the index has read, and what reading it produced. `ContentHash`,
// `Recipe` and `Producer` are empty until something computes them.
type Source struct {
	Path        string
	Kind        string
	Size        int64
	MTime       int64
	ContentHash string
	Recipe      string

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

// Chunk is one segment of a source's text.
// `Location` is the position in the source's own numbering, empty when the
// format provides none.
// `Text` is indexed and hashed but not stored; it is the text at `Start` for `Length` in the file.
// `Small` are the chunks inside this one; a Chunk with none is still large because nothing encloses it.
type Chunk struct {
	Start    int
	Length   int
	Location string
	Text     string
	Opens    []string
	Small    []Chunk
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
	return sqlwrite.InTransaction(ctx, r.db, func(tx *sqlwrite.Transaction) error {
		vault, err := vaultRow(ctx, tx, vaultID)
		if err != nil {
			return err
		}
		var row int64
		if err := tx.QueryRowContext(ctx, getStatement("save_source"),
			vault, s.Path, s.Kind, s.Size, s.MTime, newNullableValue(s.ContentHash), newNullableValue(s.Recipe), newNullableValue(s.Producer)).Scan(&row); err != nil {
			return fmt.Errorf("record the source %s: %w", s.Path, err)
		}
		return nil
	})
}

// SaveExtraction records what a file is and replaces its chunks with the ones
// its text was cut into, in one transaction.
//
// One write, because a recipe names the sizes a source's chunks were cut into:
// the recipe and the chunks it describes are recorded together.
func (r *Repository) SaveExtraction(ctx context.Context, vaultID domain.VaultID, s Source, chunks []Chunk) error {
	return sqlwrite.InTransaction(ctx, r.db, func(tx *sqlwrite.Transaction) error {
		vault, err := vaultRow(ctx, tx, vaultID)
		if err != nil {
			return err
		}
		var source int64
		if err := tx.QueryRowContext(ctx, getStatement("save_source"),
			vault, s.Path, s.Kind, s.Size, s.MTime, newNullableValue(s.ContentHash), newNullableValue(s.Recipe), newNullableValue(s.Producer)).Scan(&source); err != nil {
			return fmt.Errorf("record the source %s: %w", s.Path, err)
		}
		if err := Replace(ctx, tx, source, vault, chunks); err != nil {
			return err
		}
		return nil
	})
}

// ReplaceChunks looks up the source this vault holds at a path and replaces
// its chunks, in a transaction of its own. The source is already recorded or
// there is nothing to replace the chunks of.
func (r *Repository) ReplaceChunks(ctx context.Context, vaultID domain.VaultID, kind, path string, chunks []Chunk) error {
	return sqlwrite.InTransaction(ctx, r.db, func(tx *sqlwrite.Transaction) error {
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
		return nil
	})
}

// RemoveSources takes out the sources of one kind whose files are gone. The
// vault is authoritative: what is not on disk is not in the index.
func (r *Repository) RemoveSources(ctx context.Context, vaultID domain.VaultID, kind string, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	return sqlwrite.InTransaction(ctx, r.db, func(tx *sqlwrite.Transaction) error {
		vault, err := vaultRow(ctx, tx, vaultID)
		if err != nil {
			return err
		}
		for _, path := range paths {
			if err := removeSource(ctx, tx, vault, kind, path); err != nil {
				return err
			}
		}
		return nil
	})
}

func removeSource(ctx context.Context, tx *sqlwrite.Transaction, vault int64, kind, path string) error {
	source, err := getSourceRow(ctx, tx, vault, kind, path)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := Clear(ctx, tx, source); err != nil {
		return err
	}
	return exec(ctx, tx, "delete_source", source)
}

// MoveSources renames the source at one path, and everything under it, to a new path.
// The chunks, vectors, links and headings hang off the source's own number and
// travel with it untouched.
// Only the file at the path itself can get a new name; everything under a
// folder keeps its name.
func (r *Repository) MoveSources(ctx context.Context, vaultID domain.VaultID, from, to string) error {
	return sqlwrite.InTransaction(ctx, r.db, func(tx *sqlwrite.Transaction) error {
		vault, err := vaultRow(ctx, tx, vaultID)
		if errors.Is(err, errNoVault) {
			return nil
		}
		if err != nil {
			return err
		}
		isMoving, err := hasSourcesUnder(ctx, tx, vault, from)
		if err != nil {
			return err
		}
		if !isMoving {
			// A pass that ran between the move on disk and this call already replaced
			// the rows at from with the rows at to. Those rows are the current ones.
			return nil
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
		return nil
	})
}

func hasSourcesUnder(ctx context.Context, tx *sqlwrite.Transaction, vault int64, path string) (_ bool, err error) {
	first, past := getRangeUnder(path)
	rows, err := tx.QueryContext(ctx, getStatement("sources_under"), vault, path, first, past)
	if err != nil {
		return false, fmt.Errorf("what the vault holds at %s and under it: %w", path, err)
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	hasRow := rows.Next()
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("what the vault holds at %s and under it: %w", path, err)
	}
	return hasRow, nil
}

func deleteMoveTarget(ctx context.Context, tx *sqlwrite.Transaction, vault int64, from, to string) error {
	displaced, err := getDisplacedSources(ctx, tx, vault, from, to)
	if err != nil {
		return err
	}
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

func getDisplacedSources(
	ctx context.Context, tx *sqlwrite.Transaction, vault int64, from, to string,
) (displaced []int64, err error) {
	first, past := getRangeUnder(to)
	rows, err := tx.QueryContext(ctx, getStatement("sources_under"), vault, to, first, past)
	if err != nil {
		return nil, fmt.Errorf("what the vault holds at %s and under it: %w", to, err)
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	movingFirst, movingPast := getRangeUnder(from)
	for rows.Next() {
		var source int64
		var path string
		if err := rows.Scan(&source, &path); err != nil {
			return nil, fmt.Errorf("what the vault holds at %s and under it: %w", to, err)
		}
		if path == from || (path >= movingFirst && path < movingPast) {
			continue
		}
		displaced = append(displaced, source)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("what the vault holds at %s and under it: %w", to, err)
	}
	return displaced, nil
}

func updateNoteName(ctx context.Context, tx *sqlwrite.Transaction, vault int64, from, to string) error {
	var title string
	var named bool
	err := tx.QueryRowContext(ctx, getStatement("note_naming"), vault, from).Scan(&title, &named)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("what the note %s is called: %w", from, err)
	}
	displayName, isRenamed := domain.GetNameAfterMove(from, to, title, named)
	if !isRenamed {
		return nil
	}
	if err := exec(ctx, tx, "rename_note", domain.FoldName(domain.Basename(to)), displayName, vault, from); err != nil {
		return err
	}
	return exec(ctx, tx, "rename_title", vault, from, displayName)
}

func getRangeUnder(folder string) (first, past string) {
	return folder + "/", folder + "0"
}

func exec(ctx context.Context, tx *sqlwrite.Transaction, name string, args ...any) error {
	if _, err := tx.ExecContext(ctx, getStatement(name), args...); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("%s: %w", name, ctx.Err())
		}
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// querier runs a single-row query, on a transaction or on a plain connection.
type querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func vaultRow(ctx context.Context, db querier, identifier domain.VaultID) (int64, error) {
	var row int64
	err := db.QueryRowContext(ctx, vaultRowSQL, string(identifier)).Scan(&row)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, errNoVault
	}
	return row, err
}

func getSourceRow(ctx context.Context, db querier, vault int64, kind, path string) (int64, error) {
	var row int64
	err := db.QueryRowContext(ctx, identifySQL, vault, kind, path).Scan(&row)
	return row, err
}

func newNullableValue(s string) any {
	if s == "" {
		return nil
	}
	return s
}
