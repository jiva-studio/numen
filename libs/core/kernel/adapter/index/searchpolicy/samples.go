package searchpolicy

import (
	"context"
	"errors"
	"fmt"

	searchpolicydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/searchpolicy"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// ListSmallChunkVectors returns at most `limit` small chunks of the vault that carry a
// vector under the recipe, in the order of their content hashes.
func (r *Repository) ListSmallChunkVectors(ctx context.Context, v domain.Vault, recipe string, limit int) ([]searchpolicydomain.SampledChunk, error) {
	out, err := r.readSampledChunks(ctx, v, recipe, limit)
	if err != nil {
		return nil, fmt.Errorf("read the passages of %s: %w", v.ID, err)
	}
	return out, nil
}

func (r *Repository) readSampledChunks(ctx context.Context, v domain.Vault, recipe string, limit int) (out []searchpolicydomain.SampledChunk, err error) {
	rows, err := r.read.QueryContext(ctx, sampledChunksSQL, string(v.ID), recipe, limit)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var chunk searchpolicydomain.SampledChunk
		if err := rows.Scan(&chunk.Source, &chunk.Path, &chunk.Vector); err != nil {
			return nil, err
		}
		out = append(out, chunk)
	}
	return out, rows.Err()
}
