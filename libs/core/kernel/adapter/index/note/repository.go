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

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/chunk"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlfile"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/note/domain"
)

//go:embed sql/*.sql
var files embed.FS

func getStatement(name string) string { return sqlfile.Read(files, "sql", name) }

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

func exec(ctx context.Context, tx *sqlwrite.Transaction, name string, args ...any) error {
	if _, err := tx.ExecContext(ctx, getStatement(name), args...); err != nil {
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
	return sqlwrite.InTransaction(ctx, r.db, func(tx *sqlwrite.Transaction) error {
		vault, err := vaultRow(ctx, tx, vaultID)
		if err != nil {
			return err
		}
		for _, one := range notes {
			if err := saveNote(ctx, tx, vault, one); err != nil {
				return fmt.Errorf("%s: %w", one.Note.FileStat.Path, err)
			}
		}
		return nil
	})
}

func saveNote(
	ctx context.Context, tx *sqlwrite.Transaction, vault int64, indexed domain.NoteChunks,
) error {
	n := indexed.Note
	row, err := saveSourceAndNote(ctx, tx, vault, indexed)
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

func saveSourceAndNote(
	ctx context.Context, tx *sqlwrite.Transaction, vault int64, indexed domain.NoteChunks,
) (int64, error) {
	n := indexed.Note
	frontmatter, storeErr := encodeFrontmatter(n)
	problem := n.FrontmatterErr
	if storeErr != "" {
		// Frontmatter that cannot be stored leaves the note indexed without it,
		// and the reason is recorded as a problem of the note.
		problem = strings.TrimSpace(problem + "\n" + storeErr)
	}

	var row int64
	var updated bool
	if err := tx.QueryRowContext(ctx, getStatement("save_source"),
		vault, n.FileStat.Path, kind, n.FileStat.Size, chunk.ConvertTimeToNanos(n.FileStat.ModTime),
		nil, newNullableValue(indexed.Recipe), nil,
	).Scan(&row, &updated); err != nil {
		return 0, fmt.Errorf("record the source this note is: %w", err)
	}
	if err := exec(ctx, tx, "save_note", row, vault, domain.FoldName(domain.Basename(n.FileStat.Path)),
		n.Title, string(noteType(n)), newNullableValue(n.ID), frontmatter, newNullableValue(problem)); err != nil {
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

func insertLinks(ctx context.Context, tx *sqlwrite.Transaction, row int64, links []domain.Link) error {
	for i, l := range links {
		if _, err := tx.ExecContext(ctx, getStatement("insert_link"), row, i,
			l.Target.Scheme, l.Target.Value, domain.FoldName(domain.LinkName(l.Target.Value)),
			string(l.Role), newNullableValue(l.Type), newNullableValue(l.Why), newNullableValue(l.Label),
			newNullableValue(string(l.Marker)),
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
	return sqlwrite.InTransaction(ctx, r.db, func(tx *sqlwrite.Transaction) error {
		vault, err := vaultRow(ctx, tx, vaultID)
		if err != nil {
			return err
		}
		for _, path := range paths {
			if err := removeNote(ctx, tx, vault, path); err != nil {
				return err
			}
		}
		return nil
	})
}

func removeNote(ctx context.Context, tx *sqlwrite.Transaction, vault int64, path string) error {
	var row int64
	err := tx.QueryRowContext(ctx, getStatement("identify"), vault, path).Scan(&row)
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

func noteType(n domain.Note) domain.NoteType {
	if n.Type == "" {
		return domain.TypeNote
	}
	return n.Type
}

func newNullableValue(s string) any {
	if s == "" {
		return nil
	}
	return s
}
