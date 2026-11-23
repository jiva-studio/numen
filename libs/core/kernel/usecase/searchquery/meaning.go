package searchquery

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vector"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// vectorPage is how many vectors one read of an exact scan takes.
const vectorPage = 1000

type similarChunk struct {
	id         domain.ChunkID
	similarity float64
}

type meaningSearch struct {
	vectors  VectorQueries
	source   QuerySource
	filter   scopeFilter
	req      *searchquery.Evaluation
	warnings *warnings
}

func (m *meaningSearch) embed(ctx context.Context) (map[int][]float32, error) {
	var indices []int
	var texts []string
	for i, c := range m.req.Query.Conditions {
		if c.IsMeaning {
			indices = append(indices, i)
			texts = append(texts, c.Text)
		}
	}
	if len(texts) == 0 {
		return nil, nil
	}
	embedded, err := m.source.EmbedQueries(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("embed the meaning conditions: %w", err)
	}
	if len(embedded) != len(texts) {
		return nil, fmt.Errorf("the model returned %d vectors for %d texts", len(embedded), len(texts))
	}
	vectors := make(map[int][]float32, len(indices))
	for i, index := range indices {
		vectors[index] = embedded[i]
	}
	return vectors, nil
}

func (m *meaningSearch) findMeaning(
	ctx context.Context, index int, query []float32, documents []searchquery.Document, isListed bool,
) ([]domain.ChunkID, error) {
	limits := m.req.Limits
	if isListed && getChunkTotal(documents) <= limits.ExactScanChunks {
		found, err := m.scanVectors(ctx, documents, query)
		if err != nil {
			return nil, err
		}
		return getBest(found, limits.CandidatesPerCondition), nil
	}
	found, err := m.searchCoarse(ctx, m.req.Scopes[index], query)
	if err != nil {
		return nil, err
	}
	return getBest(found, limits.CandidatesPerCondition), nil
}

func (m *meaningSearch) scanVectors(ctx context.Context, documents []searchquery.Document, query []float32) ([]similarChunk, error) {
	if len(documents) == 0 {
		return nil, nil
	}
	var found []similarChunk
	after := domain.ChunkID("")
	for {
		page, err := m.vectors.ListDocumentVectors(ctx, m.req.Vault.ID, searchquery.DocumentVectorQuery{
			Paths: getPaths(documents), Recipe: m.req.Recipe, After: after, Limit: vectorPage,
		})
		if err != nil {
			return nil, err
		}
		similar, _ := m.keepSimilar(query, page)
		found = append(found, similar...)
		if len(page) < vectorPage {
			return found, nil
		}
		after = page[len(page)-1].Chunk
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
}

func (m *meaningSearch) searchCoarse(ctx context.Context, s searchquery.Scope, query []float32) ([]similarChunk, error) {
	limits := m.req.Limits
	limit := min(limits.CandidatesPerCondition*limits.CoarseOversample, limits.MaxCoarseCandidates)
	prefix := ""
	if len(s.PathPrefixes) == 1 {
		prefix = s.PathPrefixes[0]
	}
	for {
		ids, err := m.vectors.FindCoarseCandidates(ctx, m.req.Vault.ID, source.CoarseQuery{Recipe: m.req.Recipe, Vector: query, PathPrefix: prefix, Limit: limit})
		if err != nil {
			return nil, err
		}
		found, isPastFloor, err := m.rankCoarse(ctx, s, query, ids)
		if err != nil {
			return nil, err
		}
		if len(found) >= limits.CandidatesPerCondition || len(ids) < limit || isPastFloor {
			return found, nil
		}
		if limit >= limits.MaxCoarseCandidates {
			m.warnings.add(fmt.Sprintf(
				"the search by meaning stopped at %d nearest passages and found %d of the %d asked for", limit, len(found), limits.CandidatesPerCondition))
			return found, nil
		}
		limit = min(limit*2, limits.MaxCoarseCandidates)
	}
}

func (m *meaningSearch) rankCoarse(ctx context.Context, s searchquery.Scope, query []float32, ids []domain.ChunkID) ([]similarChunk, bool, error) {
	stored, err := m.vectors.ListStoredVectors(ctx, source.StoredVectorQuery{Recipe: m.req.Recipe, Kinds: s.SourceKinds, Chunks: ids})
	if err != nil {
		return nil, false, err
	}
	found, isPastFloor := m.keepSimilar(query, stored)
	if s.IsEveryDocument() {
		return found, isPastFloor, nil
	}
	ids = make([]domain.ChunkID, len(found))
	for i, f := range found {
		ids[i] = f.id
	}
	inScope, err := m.filter(ctx, s, ids)
	if err != nil {
		return nil, false, err
	}
	return slices.DeleteFunc(found, func(f similarChunk) bool { return !inScope[f.id] }), isPastFloor, nil
}

func (m *meaningSearch) keepSimilar(query []float32, stored []source.StoredVector) ([]similarChunk, bool) {
	var found []similarChunk
	isPastFloor := false
	for _, one := range stored {
		if len(one.Vector) != len(query) {
			continue
		}
		similarity := vector.ComputeSimilarityBytes(query, one.Vector)
		if similarity < m.req.Floor {
			isPastFloor = true
			continue
		}
		found = append(found, similarChunk{id: one.Chunk, similarity: similarity})
	}
	return found, isPastFloor
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
