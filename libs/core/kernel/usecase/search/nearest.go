package search

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"math"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vector"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// coarseCandidates is how many chunks the coarse pass keeps for each answer that leaves the
// meaning search.
//
// One bit per dimension orders by Hamming distance, and the full-precision vectors decide among
// what that order kept.
const coarseCandidates = 8

// ranked is a chunk and how near the query it is.
type ranked struct {
	chunk      domain.ChunkID
	similarity float64
}

func rankCandidates(ctx context.Context, queries sourceport.NearestQueries, q source.StoredVectorQuery, query []float32, floor float64) ([]ranked, error) {
	if len(q.Chunks) == 0 {
		return nil, nil
	}
	stored, err := queries.ListStoredVectors(ctx, q)
	if err != nil {
		return nil, err
	}
	position := make(map[domain.ChunkID]int, len(q.Chunks))
	for i, id := range q.Chunks {
		position[id] = i
	}
	passing := make([]ranked, 0, len(stored))
	for _, s := range stored {
		if len(s.Vector) != len(query) {
			return nil, fmt.Errorf("chunk %s holds %d dimensions where the query has %d", s.Chunk, len(s.Vector), len(query))
		}
		if similarity := vector.SimilarityBytes(query, s.Vector); similarity >= floor {
			passing = append(passing, ranked{chunk: s.Chunk, similarity: similarity})
		}
	}
	slices.SortFunc(passing, func(a, b ranked) int {
		if c := cmp.Compare(b.similarity, a.similarity); c != 0 {
			return c
		}
		return cmp.Compare(position[a.chunk], position[b.chunk])
	})
	return passing, nil
}

// FindNearest finds the passages of a vault nearest a query vector.
type FindNearest struct {
	queries sourceport.NearestQueries
}

// NewFindNearest creates the search by meaning over the queries of the index.
func NewFindNearest(queries sourceport.NearestQueries) FindNearest {
	return FindNearest{queries: queries}
}

// Execute returns the passages enclosing the chunks of one vault nearest the query vector, nearest
// first, at most `q.Limit` of them. The coarse pass keeps several times `q.Limit` candidates and
// the full-precision vectors order them. A chunk below the similarity floor is not a result.
// Chunks of equal similarity keep the order of the coarse pass.
func (f FindNearest) Execute(ctx context.Context, vaultID domain.VaultID, q source.NearestQuery) ([]domain.Passage, error) {
	if q.Limit <= 0 {
		return nil, fmt.Errorf("a search by meaning needs a positive limit, got %d", q.Limit)
	}
	if len(q.Vector) == 0 {
		return nil, nil
	}
	candidates, err := f.queries.FindCoarseCandidates(ctx, vaultID, source.CoarseQuery{
		Recipe: q.Recipe, Vector: q.Vector, PathPrefix: q.PathPrefix, Limit: q.Limit * coarseCandidates,
	})
	if errors.Is(err, source.ErrOtherRecipe) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	best, err := rankCandidates(ctx, f.queries, source.StoredVectorQuery{Recipe: q.Recipe, Kinds: q.Kinds, Chunks: candidates}, q.Vector, q.Floor)
	if err != nil {
		return nil, err
	}
	if len(best) > q.Limit {
		best = best[:q.Limit]
	}
	chunks := make([]domain.ChunkID, len(best))
	for i, b := range best {
		chunks[i] = b.chunk
	}
	return f.queries.GetEnclosingPassages(ctx, vaultID, chunks)
}

// GetBestSimilarity reads how near the nearest chunk of a vault is to a query vector.
type GetBestSimilarity struct {
	queries sourceport.NearestQueries
}

// NewGetBestSimilarity creates the reading over the queries of the index.
func NewGetBestSimilarity(queries sourceport.NearestQueries) GetBestSimilarity {
	return GetBestSimilarity{queries: queries}
}

// Execute returns the cosine similarity of the chunk of one vault nearest the query vector, found
// by the coarse pass and the full-precision ranking that FindNearest runs, and zero where the vault
// has none.
func (g GetBestSimilarity) Execute(ctx context.Context, vaultID domain.VaultID, recipe string, query []float32) (float64, error) {
	if len(query) == 0 {
		return 0, nil
	}
	candidates, err := g.queries.FindCoarseCandidates(ctx, vaultID, source.CoarseQuery{Recipe: recipe, Vector: query, Limit: coarseCandidates})
	if errors.Is(err, source.ErrOtherRecipe) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	best, err := rankCandidates(ctx, g.queries, source.StoredVectorQuery{Recipe: recipe, Chunks: candidates}, query, math.Inf(-1))
	if err != nil || len(best) == 0 {
		return 0, err
	}
	return best[0].similarity, nil
}
