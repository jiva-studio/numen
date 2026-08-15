// Package sqlite keeps parsed notes so they can be queried. Everything here is
// a cache: the vaults on disk are what is true, and this is what makes them
// fast to ask questions of.
package index

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	_ "modernc.org/sqlite"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// NoteRepository implements port.NoteRepository over one database file holding
// every vault.
type NoteRepository struct{ db *sql.DB }

func Open(ctx context.Context, path string) (*NoteRepository, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// WAL so a long scan does not block readers; foreign keys so removing a
	// vault cannot leave rows pointing at nothing.
	for _, pragma := range []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA foreign_keys = ON",
		"PRAGMA busy_timeout = 5000",
	} {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("%s: %w", pragma, err)
		}
	}
	if err := migrate(ctx, db); err != nil {
		db.Close()
		return nil, err
	}
	return &NoteRepository{db: db}, nil
}

func (r *NoteRepository) Close() error { return r.db.Close() }

func (r *NoteRepository) RegisterVault(ctx context.Context, v domain.Vault) error {
	_, err := r.db.ExecContext(ctx, q("register_vault"), v.ID, v.Name, v.Path)
	return err
}

func (r *NoteRepository) Known(ctx context.Context, vaultID string) (map[string]domain.FileRef, error) {
	rows, err := r.db.QueryContext(ctx, q("known_files"), vaultID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]domain.FileRef{}
	for rows.Next() {
		var ref domain.FileRef
		if err := rows.Scan(&ref.Path, &ref.Size, &ref.MTime); err != nil {
			return nil, err
		}
		out[ref.Path] = ref
	}
	return out, rows.Err()
}

func (r *NoteRepository) Put(ctx context.Context, vaultID string, n domain.Note) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, q("put_file"), vaultID, n.Ref.Path, n.Ref.Size, n.Ref.MTime); err != nil {
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
	if _, err := tx.ExecContext(ctx, q("put_note"),
		vaultID, n.Ref.Path, n.Title, frontmatter, frontmatterErr); err != nil {
		return err
	}

	for _, name := range []string{"clear_headings", "clear_tags", "clear_fts"} {
		if _, err := tx.ExecContext(ctx, q(name), vaultID, n.Ref.Path); err != nil {
			return err
		}
	}
	for _, h := range n.Headings {
		if _, err := tx.ExecContext(ctx, q("insert_heading"),
			vaultID, n.Ref.Path, h.Level, h.Text, h.Pos); err != nil {
			return err
		}
	}
	for _, tag := range n.Tags {
		if _, err := tx.ExecContext(ctx, q("insert_tag"), vaultID, n.Ref.Path, tag); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, q("insert_fts"), n.Title, n.Body, vaultID, n.Ref.Path); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *NoteRepository) Delete(ctx context.Context, vaultID string, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, path := range paths {
		for _, name := range []string{
			"clear_fts", "clear_headings", "clear_tags", "delete_note", "delete_file",
		} {
			if _, err := tx.ExecContext(ctx, q(name), vaultID, path); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func (r *NoteRepository) Search(ctx context.Context, vaultID, query string, limit int) ([]domain.Hit, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx, q("search"), query, vaultID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Hit
	for rows.Next() {
		var h domain.Hit
		if err := rows.Scan(&h.Path, &h.Title, &h.Snippet); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// Stats is what a command prints after a scan. It is a convenience of this
// adapter rather than something the core asks for.
type Stats struct{ Notes, Headings, Tags int }

func (r *NoteRepository) Stats(ctx context.Context, vaultID string) (Stats, error) {
	var s Stats
	err := r.db.QueryRowContext(ctx, q("stats"), vaultID, vaultID, vaultID).
		Scan(&s.Notes, &s.Headings, &s.Tags)
	return s, err
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
