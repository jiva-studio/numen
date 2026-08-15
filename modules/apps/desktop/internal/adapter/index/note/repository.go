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
// A note and the size and date that call it up to date are stored together or
// not at all, so an interrupted scan leaves files to be read again rather than
// rows to be trusted.
func (r *Repository) Save(ctx context.Context, vaultID string, notes []domain.Note) error {
	if len(notes) == 0 {
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
	for _, n := range notes {
		if err := saveNote(ctx, tx, vault, n); err != nil {
			return fmt.Errorf("%s: %w", n.Ref.Path, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func saveNote(ctx context.Context, tx *sql.Tx, vault int64, n domain.Note) error {
	frontmatter, storeErr := encodeFrontmatter(n)
	problem := n.FrontmatterErr
	if storeErr != "" {
		// The parser promises that a file which cannot be understood is still
		// indexed. Frontmatter that cannot be stored is the same case: the note
		// goes in without it, and the reason is recorded beside the note rather
		// than ending the scan of the whole vault.
		problem = strings.TrimSpace(problem + "\n" + storeErr)
	}

	var id int64
	if err := tx.QueryRowContext(ctx, stmt.Get("save"),
		vault, n.Ref.Path, domain.Basename(n.Ref.Path), n.Title, nullable(n.ID),
		frontmatter, nullable(problem), n.Ref.Size, n.Ref.MTime).Scan(&id); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	// Derived rows are replaced wholesale: diffing them against what was there
	// costs more than rewriting a handful of rows.
	for _, name := range []string{"clear_headings", "clear_links", "clear_problems"} {
		if err := exec(ctx, tx, name, id); err != nil {
			return err
		}
	}
	for _, h := range n.Headings {
		if err := exec(ctx, tx, "insert_heading", id, h.Pos, h.Level, h.Text); err != nil {
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
			if _, err := insert.ExecContext(ctx, id, i,
				l.Target.Scheme, l.Target.Value, domain.Basename(l.Target.Value),
				string(l.Role), nullable(l.Type), nullable(l.Note), nullable(l.Label),
			); err != nil {
				return fmt.Errorf("insert_link: %w", err)
			}
		}
	}
	for _, detail := range n.Problems {
		if err := exec(ctx, tx, "insert_problem", id, detail); err != nil {
			return err
		}
	}
	return exec(ctx, tx, "save_fts", id, n.Title, n.Body)
}

func (r *Repository) Remove(ctx context.Context, vaultID string, paths []string) error {
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
		var id int64
		err := tx.QueryRowContext(ctx, stmt.Get("identify"), vault, path).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return fmt.Errorf("identify %s: %w", path, err)
		}
		if err := exec(ctx, tx, "delete_fts", id); err != nil {
			return err
		}
		if err := exec(ctx, tx, "delete", id); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
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

// nullable keeps an empty string out of the database, so that "nothing was
// written" and "an empty value was written" stay different questions.
func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
