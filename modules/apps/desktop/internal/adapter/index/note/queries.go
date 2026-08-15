package note

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// Queries answers questions about notes in shapes that are not notes: what the
// index believes about each file, what matched a search, how much is there.
type Queries struct{ db *sql.DB }

func NewQueries(db *sql.DB) *Queries { return &Queries{db: db} }

func (q *Queries) Fingerprints(ctx context.Context, vaultID string) (map[string]domain.FileRef, error) {
	rows, err := q.db.QueryContext(ctx, stmt.Get("fingerprints"), vaultID)
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
		// knows. The caller decides; arriving here without one is a mistake in
		// the caller rather than something to paper over with a number.
		return nil, fmt.Errorf("search limit must be positive, got %d", limit)
	}
	expression := ftsExpression(query)
	if expression == "" {
		return nil, nil
	}
	rows, err := q.db.QueryContext(ctx, stmt.Get("search"), expression, vaultID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.NoteMatch
	for rows.Next() {
		var m domain.NoteMatch
		if err := rows.Scan(&m.Path, &m.Title, &m.Snippet); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (q *Queries) Summary(ctx context.Context, vaultID string) (domain.VaultSummary, error) {
	var s domain.VaultSummary
	err := q.db.QueryRowContext(ctx, stmt.Get("summary"), vaultID, vaultID).
		Scan(&s.Notes, &s.Headings)
	return s, err
}
