package searchquery

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vector"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// vectorPage is how many vectors one read of an exact scan takes.
const vectorPage = 1000

type similarChunk struct {
	id         domain.ChunkID
	similarity float64
}

// embed makes the vector of every meaning condition, in one call.
func (p *plan) embed(ctx context.Context) error {
	var indices []int
	var texts []string
	for i, c := range p.query.Conditions {
		if c.IsMeaning {
			indices = append(indices, i)
			texts = append(texts, c.Text)
		}
	}
	if len(texts) == 0 {
		return nil
	}
	vectors, err := p.u.source.EmbedQueries(ctx, texts)
	if err != nil {
		return fmt.Errorf("embed the meaning conditions: %w", err)
	}
	if len(vectors) != len(texts) {
		return fmt.Errorf("the model returned %d vectors for %d texts", len(vectors), len(texts))
	}
	p.vectors = make(map[int][]float32, len(indices))
	for i, index := range indices {
		p.vectors[index] = vectors[i]
	}
	return nil
}

// findMeaning returns the candidate chunks of a meaning condition: an exact scan over the documents of its
// scope when they are listed and few, else the approximate search, grown until enough chunks of the scope
// remain.
func (p *plan) findMeaning(ctx context.Context, index int, documents []searchquery.Document, isListed bool) ([]domain.ChunkID, error) {
	query := p.vectors[index]
	if isListed && getChunkTotal(documents) <= p.u.Limits.ExactScanChunks {
		found, err := p.scanVectors(ctx, documents, query)
		if err != nil {
			return nil, err
		}
		return getBest(found, p.u.Limits.CandidatesPerCondition), nil
	}
	found, err := p.searchCoarse(ctx, p.scopes[index], query)
	if err != nil {
		return nil, err
	}
	return getBest(found, p.u.Limits.CandidatesPerCondition), nil
}

func (p *plan) scanVectors(ctx context.Context, documents []searchquery.Document, query []float32) ([]similarChunk, error) {
	if len(documents) == 0 {
		return nil, nil
	}
	var found []similarChunk
	after := domain.ChunkID("")
	for {
		page, err := p.u.vectors.ListDocumentVectors(ctx, p.v.ID, searchquery.DocumentVectorQuery{
			Paths: getPaths(documents), Recipe: p.recipe, After: after, Limit: vectorPage,
		})
		if err != nil {
			return nil, err
		}
		found = append(found, p.keepSimilar(query, page)...)
		if len(page) < vectorPage {
			return found, nil
		}
		after = page[len(page)-1].Chunk
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
}

func (p *plan) searchCoarse(ctx context.Context, s searchquery.Scope, query []float32) ([]similarChunk, error) {
	limits := p.u.Limits
	limit := min(limits.CandidatesPerCondition*limits.CoarseOversample, limits.MaxCoarseCandidates)
	prefix := ""
	if len(s.PathPrefixes) == 1 {
		prefix = s.PathPrefixes[0]
	}
	for {
		ids, err := p.u.vectors.FindCoarseCandidates(ctx, p.v.ID, source.CoarseQuery{Recipe: p.recipe, Vector: query, PathPrefix: prefix, Limit: limit})
		if err != nil {
			return nil, err
		}
		found, err := p.rankCoarse(ctx, s, query, ids)
		if err != nil {
			return nil, err
		}
		if len(found) >= limits.CandidatesPerCondition || len(ids) < limit {
			return found, nil
		}
		if limit >= limits.MaxCoarseCandidates {
			p.warnings = append(p.warnings, fmt.Sprintf(
				"the search by meaning stopped at %d nearest passages and found %d of the %d asked for", limit, len(found), limits.CandidatesPerCondition))
			return found, nil
		}
		limit = min(limit*2, limits.MaxCoarseCandidates)
	}
}

func (p *plan) rankCoarse(ctx context.Context, s searchquery.Scope, query []float32, ids []domain.ChunkID) ([]similarChunk, error) {
	stored, err := p.u.vectors.ListStoredVectors(ctx, source.StoredVectorQuery{Recipe: p.recipe, Kinds: s.SourceKinds, Chunks: ids})
	if err != nil {
		return nil, err
	}
	found := p.keepSimilar(query, stored)
	if s.IsEveryDocument() {
		return found, nil
	}
	ids = make([]domain.ChunkID, len(found))
	for i, f := range found {
		ids[i] = f.id
	}
	inScope, err := p.filterScope(ctx, s, ids)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(found, func(f similarChunk) bool { return !inScope[f.id] }), nil
}

func (p *plan) keepSimilar(query []float32, stored []source.StoredVector) []similarChunk {
	var found []similarChunk
	for _, one := range stored {
		if len(one.Vector) != len(query) {
			continue
		}
		if similarity := vector.SimilarityBytes(query, one.Vector); similarity >= p.floor {
			found = append(found, similarChunk{id: one.Chunk, similarity: similarity})
		}
	}
	return found
}

func getBest(found []similarChunk, limit int) []domain.ChunkID {
	slices.SortStableFunc(found, func(a, b similarChunk) int {
		return cmp.Or(cmp.Compare(b.similarity, a.similarity), compareChunkIDs(a.id, b.id))
	})
	ids := make([]domain.ChunkID, 0, min(len(found), limit))
	for _, f := range found[:min(len(found), limit)] {
		ids = append(ids, f.id)
	}
	return ids
}

func compareChunkIDs(a, b domain.ChunkID) int {
	return cmp.Or(cmp.Compare(len(a), len(b)), cmp.Compare(a, b))
}
