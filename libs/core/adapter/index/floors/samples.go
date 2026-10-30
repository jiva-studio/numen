package floors

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/port/floorport"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// Samples reads the titles and vectors a calibration measures.
type Samples struct{ db *sql.DB }

// NewSamples is the samples over a pool that reads.
func NewSamples(db *sql.DB) *Samples { return &Samples{db: db} }

// GetTitles returns at most `limit` note titles of the vault, in the order of
// their content hashes.
func (s *Samples) GetTitles(ctx context.Context, v domain.Vault, limit int) ([]floorport.Title, error) {
	out, err := s.readTitles(ctx, v, limit)
	if err != nil {
		return nil, fmt.Errorf("read the titles of %s: %w", v.ID, err)
	}
	return out, nil
}

func (s *Samples) readTitles(ctx context.Context, v domain.Vault, limit int) (out []floorport.Title, err error) {
	rows, err := s.db.QueryContext(ctx, stmt.Get("titles"), string(v.ID), limit)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var title floorport.Title
		if err := rows.Scan(&title.Source, &title.Text); err != nil {
			return nil, err
		}
		out = append(out, title)
	}
	return out, rows.Err()
}

// GetPassages returns at most `limit` small chunks of the vault that carry a
// vector under the recipe, in the order of their content hashes.
func (s *Samples) GetPassages(ctx context.Context, v domain.Vault, recipe string, limit int) ([]floorport.Passage, error) {
	out, err := s.readPassages(ctx, v, recipe, limit)
	if err != nil {
		return nil, fmt.Errorf("read the passages of %s: %w", v.ID, err)
	}
	return out, nil
}

func (s *Samples) readPassages(ctx context.Context, v domain.Vault, recipe string, limit int) (out []floorport.Passage, err error) {
	rows, err := s.db.QueryContext(ctx, stmt.Get("passages"), string(v.ID), recipe, limit)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var passage floorport.Passage
		if err := rows.Scan(&passage.Source, &passage.Vector); err != nil {
			return nil, err
		}
		out = append(out, passage)
	}
	return out, rows.Err()
}
