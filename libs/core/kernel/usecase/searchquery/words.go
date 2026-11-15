package searchquery

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/note/domain"
)

const messageInvalidWords = "words takes an FTS5 expression, and this one is not valid"

func (p *plan) findWords(ctx context.Context, index int, documents []searchquery.Document, isListed bool) ([]domain.ChunkID, error) {
	q, ok := p.newWordQuery(index, documents, isListed)
	if !ok {
		return nil, nil
	}
	s := p.scopes[index]
	limits := p.u.Limits
	for {
		matches, err := p.u.index.words.FindWords(ctx, p.v.ID, q)
		if err != nil {
			return nil, p.getWordsError(index, err)
		}
		ids := getMatchChunks(matches)
		if s.IsEveryDocument() {
			return ids, nil
		}
		kept, err := p.keepInScope(ctx, s, ids)
		if err != nil {
			return nil, err
		}
		if len(kept) >= limits.CandidatesPerCondition || len(matches) < q.Limit {
			return kept[:min(len(kept), limits.CandidatesPerCondition)], nil
		}
		if q.Limit >= limits.MaxCoarseCandidates {
			p.warnings = append(p.warnings, fmt.Sprintf(
				"the search by words stopped at %d best passages and found %d of the %d asked for", q.Limit, len(kept), limits.CandidatesPerCondition))
			return kept, nil
		}
		q.Limit = min(q.Limit*2, limits.MaxCoarseCandidates)
	}
}

func (p *plan) newWordQuery(index int, documents []searchquery.Document, isListed bool) (searchquery.WordQuery, bool) {
	limits := p.u.Limits
	q := searchquery.WordQuery{Expression: p.query.Conditions[index].Text, Limit: limits.CandidatesPerCondition}
	if !isListed || getChunkTotal(documents) > limits.ExactScanChunks {
		s := p.scopes[index]
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

func (p *plan) keepInScope(ctx context.Context, s searchquery.Scope, ids []domain.ChunkID) ([]domain.ChunkID, error) {
	inScope, err := p.filterScope(ctx, s, ids)
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

func (p *plan) getWordsError(index int, err error) error {
	if !errors.Is(err, source.ErrInvalidWords) {
		return err
	}
	position, _ := findConditionPosition(p.query.Root, index)
	return &searchquery.ParseError{Problems: []searchquery.Problem{{Position: position, Message: messageInvalidWords}}}
}

func findConditionPosition(node searchquery.Node, index int) (searchquery.Position, bool) {
	if node.Operator == searchquery.OperatorLeaf {
		return node.Leaf.Position, node.Leaf.IsChunkLevel() && node.Leaf.ConditionIndex == index
	}
	for _, child := range node.Children {
		if position, found := findConditionPosition(child, index); found {
			return position, true
		}
	}
	return searchquery.StartPosition(), false
}
