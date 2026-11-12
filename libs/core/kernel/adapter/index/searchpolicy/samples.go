package searchpolicy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	searchpolicydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/searchpolicy"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// Samples reads the vectors a plugin measures a vault with.
type Samples struct{ db *sql.DB }

// NewSamples is the samples over a pool that reads.
func NewSamples(db *sql.DB) *Samples { return &Samples{db: db} }

// ListSmallChunkVectors returns at most `limit` small chunks of the vault that carry a
// vector under the recipe, in the order of their content hashes.
func (s *Samples) ListSmallChunkVectors(ctx context.Context, v domain.Vault, recipe string, limit int) ([]searchpolicydomain.Passage, error) {
	out, err := s.readPassages(ctx, v, recipe, limit)
	if err != nil {
		return nil, fmt.Errorf("read the passages of %s: %w", v.ID, err)
	}
	return out, nil
}

func (s *Samples) readPassages(ctx context.Context, v domain.Vault, recipe string, limit int) (out []searchpolicydomain.Passage, err error) {
	rows, err := s.db.QueryContext(ctx, passagesSQL, string(v.ID), recipe, limit)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var passage searchpolicydomain.Passage
		if err := rows.Scan(&passage.Source, &passage.Path, &passage.Vector); err != nil {
			return nil, err
		}
		out = append(out, passage)
	}
	return out, rows.Err()
}
