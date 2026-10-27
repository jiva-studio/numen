// Package note stores notes and answers questions about them. The repository
// and the queries are separate types on purpose: a repository is a collection of
// notes, and a search result or a count is not a note.
package note

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/core/adapter/index/chunk"
	"github.com/jiva-studio/numen/libs/core/adapter/index/sqlfile"
	"github.com/jiva-studio/numen/libs/core/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

//go:embed sql/*.sql
var files embed.FS

var stmt = sqlfile.Load(files, "sql")

// kind is what a note is among the sources the index holds. Everything this
// package writes and reads is one, and a path naming a file of another kind
// answers nothing here.
const kind = "note"

// Repository is the collection of notes. It puts one in and takes one out, and
// answers no questions about them.
type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

// exec runs a named statement and says which one failed. A bare driver error
// from one of the many statements in a transaction is a schema mistake nobody
// can locate.
func exec(ctx context.Context, tx *sqlwrite.Transaction, name string, args ...any) error {
	if _, err := tx.ExecContext(ctx, stmt.Get(name), args...); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// Save writes a group of notes in one transaction.
//
// A note and the size and date that call it up to date are stored together or
// not at all, so an interrupted scan leaves files to be read again.
func (r *Repository) Save(
	ctx context.Context, vaultID domain.VaultID, notes []domain.NoteChunks,
) error {
	if len(notes) == 0 {
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
	for _, one := range notes {
		if err := saveNote(ctx, tx, vault, one); err != nil {
			return fmt.Errorf("%s: %w", one.Note.FileStat.Path, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func saveNote(
	ctx context.Context, tx *sqlwrite.Transaction, vault int64, indexed domain.NoteChunks,
) error {
	n := indexed.Note
	row, err := saveSourceAndNote(ctx, tx, vault, n)
	if err != nil {
		return err
	}
	// A chunk whose text is what it was keeps its row, and the vector made from it.
	if err := chunk.Replace(ctx, tx, row, vault, chunk.FromDomain(indexed.Chunks)); err != nil {
		return err
	}
	for _, h := range indexed.Headings {
		if err := exec(ctx, tx, "insert_heading", row, h.Line, h.Level, h.Text); err != nil {
			return err
		}
	}

	// The names this note answers to, indexed for the words in them: its own
	// title, and every heading inside it. The headings go in once they are all
	// stored, because the index of them is keyed by the numbers they were
	// stored under.
	if err := exec(ctx, tx, "insert_title", row, n.Title); err != nil {
		return err
	}
	if err := exec(ctx, tx, "insert_heading_names", row); err != nil {
		return err
	}
	if err := insertLinks(ctx, tx, row, n.Links); err != nil {
		return err
	}
	for _, detail := range n.Problems {
		if err := exec(ctx, tx, "insert_problem", row, detail); err != nil {
			return err
		}
	}
	return nil
}

// saveSourceAndNote stores the source and note rows and gives the source row,
// clearing the derived rows of a source that already existed.
func saveSourceAndNote(
	ctx context.Context, tx *sqlwrite.Transaction, vault int64, n domain.Note,
) (int64, error) {
	frontmatter, storeErr := encodeFrontmatter(n)
	problem := n.FrontmatterErr
	if storeErr != "" {
		// Frontmatter that cannot be stored leaves the note indexed without it,
		// and the reason is recorded as a problem of the note.
		problem = strings.TrimSpace(problem + "\n" + storeErr)
	}

	var row int64
	var updated bool
	if err := tx.QueryRowContext(ctx, stmt.Get("save_source"),
		vault, n.FileStat.Path, kind, n.FileStat.Size, chunk.ConvertTimeToNanos(n.FileStat.ModTime),
		nil, nil, nil,
	).Scan(&row, &updated); err != nil {
		return 0, fmt.Errorf("record the source this note is: %w", err)
	}
	if err := exec(ctx, tx, "save_note", row, vault, domain.FoldName(domain.Basename(n.FileStat.Path)),
		n.Title, string(noteType(n)), newNullable(n.ID), frontmatter, newNullable(problem)); err != nil {
		return 0, err
	}
	if !updated {
		return row, nil
	}
	for _, name := range []string{
		"clear_heading_names", "clear_headings", "clear_links", "clear_problems",
	} {
		if err := exec(ctx, tx, name, row); err != nil {
			return 0, err
		}
	}
	return row, nil
}

// insertLinks stores what a note points at, in the order it names them.
func insertLinks(ctx context.Context, tx *sqlwrite.Transaction, row int64, links []domain.Link) error {
	for i, l := range links {
		if _, err := tx.ExecContext(ctx, stmt.Get("insert_link"), row, i,
			l.Target.Scheme, l.Target.Value, domain.FoldName(domain.LinkName(l.Target.Value)),
			string(l.Role), newNullable(l.Type), newNullable(l.Why), newNullable(l.Label),
		); err != nil {
			return fmt.Errorf("store what this note points at: %w", err)
		}
	}
	return nil
}

func (r *Repository) Remove(ctx context.Context, vaultID domain.VaultID, paths []string) error {
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
		if err := removeNote(ctx, tx, vault, path); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// removeNote deletes the note at a path, and its rows in the virtual tables.
func removeNote(ctx context.Context, tx *sqlwrite.Transaction, vault int64, path string) error {
	var row int64
	err := tx.QueryRowContext(ctx, stmt.Get("identify"), vault, path).Scan(&row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("identify %s: %w", path, err)
	}
	if err := chunk.Clear(ctx, tx, row); err != nil {
		return err
	}
	for _, name := range []string{"clear_heading_names", "delete_title_name"} {
		if err := exec(ctx, tx, name, row); err != nil {
			return err
		}
	}
	return exec(ctx, tx, "delete_source", row)
}

// encodeFrontmatter stores what was found as JSON, or nothing when there was no
// frontmatter at all — which is different from an empty one.
//
// YAML can hold things JSON cannot: a NaN, a non-string key, a value that
// refers to itself. The note is still a note, so a failure here comes back as
// a reason, and the caller records it beside the note.
func encodeFrontmatter(n domain.Note) (value any, problem string) {
	if n.Frontmatter == nil {
		return nil, ""
	}
	raw, err := json.Marshal(n.Frontmatter)
	if err != nil {
		return nil, "frontmatter could not be stored: " + err.Error()
	}
	return string(raw), ""
}

// noteType is what the file said it is. A note whose file says nothing is a
// note, and the column holds one of the four words either way.
func noteType(n domain.Note) domain.NoteType {
	if n.Type == "" {
		return domain.TypeNote
	}
	return n.Type
}

// newNullable keeps an empty string out of the database, so that "nothing was
// written" and "an empty value was written" stay different questions.
func newNullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
