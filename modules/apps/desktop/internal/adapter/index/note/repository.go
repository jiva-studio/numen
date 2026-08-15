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

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/index/sqlfile"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

//go:embed sql/*.sql
var files embed.FS

var stmt = sqlfile.Load(files, "sql")

// Repository is the collection of notes. It puts one in and takes one out, and
// answers no questions about them.
type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

// exec runs a named statement and says which one failed. A bare driver error
// from one of six statements in a transaction is a schema mistake nobody can
// locate.
func exec(ctx context.Context, tx *sql.Tx, name string, args ...any) error {
	if _, err := tx.ExecContext(ctx, stmt.Get(name), args...); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// Save writes a group of notes in one transaction.
//
// One transaction rather than one per note: a commit writes to the log and then
// walks the page cache, and that work is the same size whether one note or five
// hundred went into it. It is also what makes a interrupted scan harmless —
// either a note and the fingerprint that dates it are both stored, or neither
// is, so the next scan reads the file again instead of trusting a row that was
// never finished.
func (r *Repository) Save(ctx context.Context, vaultID string, notes []domain.Note) error {
	if len(notes) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, n := range notes {
		if err := saveNote(ctx, tx, vaultID, n); err != nil {
			return fmt.Errorf("%s: %w", n.Ref.Path, err)
		}
	}
	return tx.Commit()
}

func saveNote(ctx context.Context, tx *sql.Tx, vaultID string, n domain.Note) error {
	if err := exec(ctx, tx, "save_file", vaultID, n.Ref.Path, n.Ref.Size, n.Ref.MTime); err != nil {
		return err
	}

	frontmatter, storeErr := encodeFrontmatter(n)
	problem := n.FrontmatterErr
	if storeErr != "" {
		// The parser promises that a file which cannot be understood is still
		// indexed. Frontmatter that cannot be stored is the same case: the note
		// goes in without it, and the reason is recorded beside the note rather
		// than ending the scan of the whole vault.
		problem = strings.TrimSpace(problem + "\n" + storeErr)
	}
	var frontmatterErr any
	if problem != "" {
		frontmatterErr = problem
	}
	if err := exec(ctx, tx, "save", vaultID, n.Ref.Path, n.Title, frontmatter, frontmatterErr,
		nullable(n.ID), basename(n.Ref.Path)); err != nil {
		return err
	}

	// The full-text row is addressed by the note's own rowid. An FTS5 table has
	// no other key: matching on the columns instead scans the entire index, and
	// doing that once per note saved makes a rebuild quadratic.
	rowID, err := noteRowID(ctx, tx, vaultID, n.Ref.Path)
	if err != nil {
		return err
	}

	// Derived rows are replaced wholesale: diffing them against what was there
	// costs more than rewriting a handful of rows.
	for _, name := range []string{"clear_headings", "clear_links", "clear_problems"} {
		if err := exec(ctx, tx, name, vaultID, n.Ref.Path); err != nil {
			return err
		}
	}
	if err := exec(ctx, tx, "clear_fts", rowID); err != nil {
		return err
	}
	for _, h := range n.Headings {
		if _, err := tx.ExecContext(ctx, stmt.Get("insert_heading"),
			vaultID, n.Ref.Path, h.Level, h.Text, h.Pos); err != nil {
			return err
		}
	}
	if len(n.Links) > 0 {
		insert, err := tx.PrepareContext(ctx, stmt.Get("insert_link"))
		if err != nil {
			return fmt.Errorf("insert_link: %w", err)
		}
		defer insert.Close()
		for i, l := range n.Links {
			if _, err := insert.ExecContext(ctx, vaultID, n.Ref.Path,
				l.Target.Scheme, l.Target.Value, string(l.Role),
				nullable(l.Type), nullable(l.Note), nullable(l.Label),
				basename(l.Target.Value), i); err != nil {
				return fmt.Errorf("insert_link: %w", err)
			}
		}
	}
	for _, detail := range n.Problems {
		if err := exec(ctx, tx, "insert_problem", vaultID, n.Ref.Path, detail); err != nil {
			return err
		}
	}
	return exec(ctx, tx, "insert_fts", rowID, n.Title, n.Body, vaultID, n.Ref.Path)
}

func (r *Repository) Remove(ctx context.Context, vaultID string, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, path := range paths {
		rowID, err := noteRowID(ctx, tx, vaultID, path)
		if err != nil {
			return err
		}
		if err := exec(ctx, tx, "clear_fts", rowID); err != nil {
			return err
		}
		for _, name := range []string{"clear_headings", "clear_links", "clear_problems", "delete", "delete_file"} {
			if err := exec(ctx, tx, name, vaultID, path); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

// encodeFrontmatter stores what was found as JSON, or nothing when there was no
// frontmatter at all — which is different from an empty one.
//
// YAML can hold things JSON cannot: a NaN, a non-string key, a value that
// refers to itself. The note is still a note, so a failure here returns the
// reason rather than an error, and the caller records it beside the note.
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

// noteRowID is the identity of a note inside this database, and therefore the
// identity of its full-text row. A note that has never been saved has none yet,
// in which case there is nothing to clear.
func noteRowID(ctx context.Context, tx *sql.Tx, vaultID, path string) (int64, error) {
	var rowID int64
	err := tx.QueryRowContext(ctx, stmt.Get("rowid"), vaultID, path).Scan(&rowID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return rowID, err
}

// nullable keeps an empty string out of the database, so that "nothing was
// written" and "an empty value was written" stay different questions.
func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// basename is the name a note is found by when a link is written by name.
func basename(path string) string {
	name := path
	if i := strings.LastIndexByte(name, '/'); i >= 0 {
		name = name[i+1:]
	}
	if i := strings.LastIndexByte(name, '.'); i > 0 {
		name = name[:i]
	}
	return name
}
