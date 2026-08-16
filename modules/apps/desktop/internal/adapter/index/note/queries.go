package note

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// Queries answers questions about notes in shapes that are not notes: what the
// index believes about each file, what matched a search, how much is there.
type Queries struct{ db *sql.DB }

func NewQueries(db *sql.DB) *Queries { return &Queries{db: db} }

func (q *Queries) Fingerprints(ctx context.Context, vaultID string) (map[string]domain.FileRef, error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return map[string]domain.FileRef{}, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := q.db.QueryContext(ctx, stmt.Get("fingerprints"), vault)
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

func (q *Queries) Search(ctx context.Context, vaultID, query string, limit int) ([]domain.NoteMatch, error) {
	if limit <= 0 {
		// How many results a person wants is not something a database adapter
		// knows. The caller decides, and arriving here without one is a
		// mistake in the caller.
		return nil, fmt.Errorf("search limit must be positive, got %d", limit)
	}
	expression := ftsExpression(query)
	if expression == "" {
		return nil, nil
	}
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := q.db.QueryContext(ctx, stmt.Get("search"), expression, vault, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.NoteMatch
	for rows.Next() {
		var m domain.NoteMatch
		if err := rows.Scan(&m.Path, &m.Title); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (q *Queries) Summary(ctx context.Context, vaultID string) (domain.VaultSummary, error) {
	var s domain.VaultSummary
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	err = q.db.QueryRowContext(ctx, stmt.Get("summary"), vault, vault).Scan(&s.Notes, &s.Headings)
	return s, err
}

// Statements exposes the SQL this package runs, so that a test can ask the
// database how it intends to answer each one. A plan is not something a package
// can check about itself: it needs a migrated database, and that lives one level
// up.
func Statements() map[string]string { return stmt }

// Named is every note filed under one name. A name that answers for more than
// one note is what makes a link written by that name ambiguous, and
// is worth saying out loud before it surprises anyone.
func (q *Queries) Named(ctx context.Context, vaultID, name string) ([]string, error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := q.db.QueryContext(ctx, stmt.Get("named"), vault, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		out = append(out, path)
	}
	return out, rows.Err()
}
