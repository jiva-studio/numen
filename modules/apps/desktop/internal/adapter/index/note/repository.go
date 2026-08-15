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

func (r *Repository) Save(ctx context.Context, vaultID string, n domain.Note) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, stmt.Get("save_file"),
		vaultID, n.Ref.Path, n.Ref.Size, n.Ref.MTime); err != nil {
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
	if _, err := tx.ExecContext(ctx, stmt.Get("save"),
		vaultID, n.Ref.Path, n.Title, frontmatter, frontmatterErr); err != nil {
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
	if _, err := tx.ExecContext(ctx, stmt.Get("clear_headings"), vaultID, n.Ref.Path); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, stmt.Get("clear_fts"), rowID); err != nil {
		return err
	}
	for _, h := range n.Headings {
		if _, err := tx.ExecContext(ctx, stmt.Get("insert_heading"),
			vaultID, n.Ref.Path, h.Level, h.Text, h.Pos); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, stmt.Get("insert_fts"),
		rowID, n.Title, n.Body, vaultID, n.Ref.Path); err != nil {
		return err
	}
	return tx.Commit()
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
		if _, err := tx.ExecContext(ctx, stmt.Get("clear_fts"), rowID); err != nil {
			return err
		}
		for _, name := range []string{"clear_headings", "delete", "delete_file"} {
			if _, err := tx.ExecContext(ctx, stmt.Get(name), vaultID, path); err != nil {
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
