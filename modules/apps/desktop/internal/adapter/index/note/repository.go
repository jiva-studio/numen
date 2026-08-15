// Package note stores notes and answers questions about them. The repository
// and the queries are separate types on purpose: a repository is a collection of
// notes, and a search result or a count is not a note.
package note

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"

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

	frontmatter, err := encodeFrontmatter(n)
	if err != nil {
		return err
	}
	var frontmatterErr any
	if n.FrontmatterErr != "" {
		frontmatterErr = n.FrontmatterErr
	}
	if _, err := tx.ExecContext(ctx, stmt.Get("save"),
		vaultID, n.Ref.Path, n.Title, frontmatter, frontmatterErr); err != nil {
		return err
	}

	// Derived rows are replaced wholesale: diffing them against what was there
	// costs more than rewriting a handful of rows.
	for _, name := range []string{"clear_headings", "clear_fts"} {
		if _, err := tx.ExecContext(ctx, stmt.Get(name), vaultID, n.Ref.Path); err != nil {
			return err
		}
	}
	for _, h := range n.Headings {
		if _, err := tx.ExecContext(ctx, stmt.Get("insert_heading"),
			vaultID, n.Ref.Path, h.Level, h.Text, h.Pos); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, stmt.Get("insert_fts"),
		n.Title, n.Body, vaultID, n.Ref.Path); err != nil {
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
		for _, name := range []string{"clear_fts", "clear_headings", "delete", "delete_file"} {
			if _, err := tx.ExecContext(ctx, stmt.Get(name), vaultID, path); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

// encodeFrontmatter stores what was found as JSON, or nothing when there was no
// frontmatter at all — which is different from an empty one.
func encodeFrontmatter(n domain.Note) (any, error) {
	if n.Frontmatter == nil {
		return nil, nil
	}
	raw, err := json.Marshal(n.Frontmatter)
	if err != nil {
		return nil, err
	}
	return string(raw), nil
}
