package search

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchpolicy"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/vector"
)

// Samples reads the small chunks of a vault that carry a vector. The order is a function of the content
// hashes in the index, so the same index yields the same sample.
type Samples interface {
	// ListSmallChunkVectors returns at most `limit` small chunks of the vault that carry a
	// vector under the recipe.
	ListSmallChunkVectors(ctx context.Context, v domain.Vault, recipe string, limit int) ([]searchpolicy.Passage, error)
}

// SampleSimilarities compares each query with the sampled small chunks of the vault. At most
// MaxSampledQueries queries are taken, else ErrLimitExceeded. A query whose dimension count differs
// from a passage's returns ErrInvalidQuery.
func SampleSimilarities(
	ctx context.Context, samples Samples, v domain.Vault, recipe string, queries []plugin.SimilarityQuery,
) (plugin.SimilaritySample, error) {
	if len(queries) > plugin.MaxSampledQueries {
		return plugin.SimilaritySample{}, fmt.Errorf("%w: %d queries, at most %d", plugin.ErrLimitExceeded, len(queries), plugin.MaxSampledQueries)
	}
	passages, err := samples.ListSmallChunkVectors(ctx, v, recipe, plugin.MaxSampledPassages)
	if err != nil {
		return plugin.SimilaritySample{}, err
	}
	sources := make(map[int64]struct{}, len(passages))
	for _, passage := range passages {
		sources[passage.Source] = struct{}{}
	}
	rows := make([][]float64, len(queries))
	for i, query := range queries {
		if err := ctx.Err(); err != nil {
			return plugin.SimilaritySample{}, err
		}
		row, err := compare(query, passages)
		if err != nil {
			return plugin.SimilaritySample{}, fmt.Errorf("query %d: %w", i, err)
		}
		rows[i] = row
	}
	return plugin.SimilaritySample{PassageCount: len(passages), SourceCount: len(sources), Rows: rows}, nil
}

func compare(query plugin.SimilarityQuery, passages []searchpolicy.Passage) ([]float64, error) {
	if len(query.Vector) == 0 {
		return nil, nil
	}
	row := make([]float64, 0, len(passages))
	for _, passage := range passages {
		if query.Path != "" && passage.Path == query.Path {
			continue
		}
		if len(passage.Vector) != len(query.Vector) {
			return nil, fmt.Errorf("%w: a passage holds %d dimensions where the query has %d", plugin.ErrInvalidQuery, len(passage.Vector), len(query.Vector))
		}
		row = append(row, vector.SimilarityBytes(query.Vector, passage.Vector))
	}
	return row, nil
}
