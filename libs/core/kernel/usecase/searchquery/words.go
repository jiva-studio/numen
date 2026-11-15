package searchquery

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/note/domain"
)

const messageInvalidWords = "words takes an FTS5 expression, and this one is not valid"

func (p *plan) findWords(ctx context.Context, index int, documents []searchquery.Document, isListed bool) ([]domain.ChunkID, error) {
	s := p.scopes[index]
	q := searchquery.WordQuery{Expression: p.query.Conditions[index].Text, Limit: p.u.Limits.CandidatesPerCondition}
	if isListed && getChunkTotal(documents) <= p.u.Limits.ExactScanChunks {
		if len(documents) == 0 {
			return nil, nil
		}
		q.Paths = getPaths(documents)
	} else {
		q.PathPrefixes, q.SourceKinds = s.PathPrefixes, s.SourceKinds
	}
	matches, err := p.u.words.FindWords(ctx, p.v.ID, q)
	if err != nil {
		return nil, p.getWordsError(index, err)
	}
	ids := make([]domain.ChunkID, len(matches))
	for i, m := range matches {
		ids[i] = m.Chunk
	}
	if s.IsEveryDocument() {
		return ids, nil
	}
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
