package searchquery

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

const messageInvalidWords = "words takes an FTS5 expression, and this one is not valid"

type wordSearch struct {
	words    WordQueries
	filter   scopeFilter
	req      *searchquery.Evaluation
	warnings *warnings
}

func (w *wordSearch) findWords(ctx context.Context, index int, documents []searchquery.Document, isListed bool) ([]domain.ChunkID, error) {
	q, ok := w.newWordQuery(index, documents, isListed)
	if !ok {
		return nil, nil
	}
	s := w.req.Scopes[index]
	limits := w.req.Limits
	for {
		matches, err := w.words.FindWords(ctx, w.req.Vault.ID, q)
		if err != nil {
			return nil, getWordsError(w.req, index, err)
		}
		ids := getMatchChunks(matches)
		if s.IsEveryDocument() {
			return ids, nil
		}
		kept, err := w.keepInScope(ctx, s, ids)
		if err != nil {
			return nil, err
		}
		if len(kept) >= limits.CandidatesPerCondition || len(matches) < q.Limit {
			return kept[:min(len(kept), limits.CandidatesPerCondition)], nil
		}
		if q.Limit >= limits.MaxCoarseCandidates {
			w.warnings.add(fmt.Sprintf(
				"the search by words stopped at %d best passages and found %d of the %d asked for", q.Limit, len(kept), limits.CandidatesPerCondition))
			return kept, nil
		}
		q.Limit = min(q.Limit*2, limits.MaxCoarseCandidates)
	}
}

func (w *wordSearch) newWordQuery(index int, documents []searchquery.Document, isListed bool) (searchquery.WordQuery, bool) {
	limits := w.req.Limits
	q := searchquery.WordQuery{Expression: w.req.Query.Conditions[index].Text, Limit: limits.CandidatesPerCondition}
	if !isListed || getChunkTotal(documents) > limits.ExactScanChunks {
		s := w.req.Scopes[index]
		q.PathPrefixes, q.SourceKinds = s.PathPrefixes, s.SourceKinds
		return q, true
	}
	if len(documents) == 0 {
		return q, false
	}
	q.Paths = getPaths(documents)
	return q, true
}

func getMatchChunks(matches []searchquery.WordMatch) []domain.ChunkID {
	ids := make([]domain.ChunkID, len(matches))
	for i, m := range matches {
		ids[i] = m.Chunk
	}
	return ids
}

func (w *wordSearch) keepInScope(ctx context.Context, s searchquery.Scope, ids []domain.ChunkID) ([]domain.ChunkID, error) {
	inScope, err := w.filter(ctx, s, ids)
	if err != nil {
		return nil, err
	}
	kept := ids[:0]
	for _, id := range ids {
		if inScope[id] {
			kept = append(kept, id)
		}
	}
	return kept, nil
}
