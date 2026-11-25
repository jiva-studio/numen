package search

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
	sourcedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
)

// Words ranks the chunks of a vault by the words of the query, through the full text index.
type Words struct {
	queries LexicalQueries
}

// NewWords returns the participant over the word queries of the index.
func NewWords(queries LexicalQueries) Words {
	return Words{queries: queries}
}

// Rank returns the chunks that hold the words of the query, best first, each with its word score.
func (w Words) Rank(ctx context.Context, v domain.Vault, q Query) (ParticipantResults, error) {
	found, err := w.queries.FindLexical(ctx, v.ID, getLexicalQuery(q))
	return ParticipantResults{Results: found}, err
}

// Names ranks the sections of a vault whose names match the query. Its ranking leads: the section a
// source is found by comes before the other search results of that source.
type Names struct {
	queries LexicalQueries
}

// NewNames returns the participant over the section names of the index.
func NewNames(queries LexicalQueries) Names {
	return Names{queries: queries}
}

// Rank returns the sections whose names match the query, best first.
func (n Names) Rank(ctx context.Context, v domain.Vault, q Query) (ParticipantResults, error) {
	found, err := n.queries.GetNamedSections(ctx, v.ID, getLexicalQuery(q))
	return ParticipantResults{Results: found, IsLeading: true}, err
}

func getLexicalQuery(q Query) sourcedomain.LexicalQuery {
	return sourcedomain.LexicalQuery{Text: q.Text, Kinds: q.Kinds, Limit: q.Limit, IsGrowing: q.IsGrowing, PathPrefix: q.PathPrefix}
}

// EnclosingResults reads the search results that enclose chunks of a vault.
type EnclosingResults interface {
	// GetEnclosingResults returns the search result that encloses each chunk of one vault, in the order
	// of `chunks`, leaving out a chunk the vault does not hold.
	GetEnclosingResults(ctx context.Context, vaultID domain.VaultID, chunks []domain.ChunkID) ([]domain.SearchResult, error)
}

// Source is a search source of a plugin as a participant: it asks the source for the chunks of its place
// and turns them into the search results that enclose them.
type Source struct {
	source    searchsourceport.Source
	enclosing EnclosingResults
}

// NewSource returns the participant over a search source and the search results of the index.
func NewSource(source searchsourceport.Source, enclosing EnclosingResults) Source {
	return Source{source: source, enclosing: enclosing}
}

// Rank returns the search results enclosing the chunks the source answers, in the order of the source, each
// with the similarity the source scored its chunk with. A source that has not described itself keeps no vectors
// and answers nothing.
func (s Source) Rank(ctx context.Context, v domain.Vault, q Query) (ParticipantResults, error) {
	described := s.source.Describe()
	if !described.IsDescribed() {
		return ParticipantResults{}, nil
	}
	place := searchsource.Place{VaultID: v.ID, Recipe: described.Recipe()}
	matches, err := s.source.SearchChunks(ctx, place, searchsource.Query{
		Text: q.Text, Kinds: q.Kinds, PathPrefix: q.PathPrefix, Limit: q.Limit, Floor: q.Floor,
	})
	if err != nil || len(matches) == 0 {
		return ParticipantResults{}, err
	}
	chunks := make([]domain.ChunkID, len(matches))
	for i, one := range matches {
		chunks[i] = one.ChunkID
	}
	found, err := s.enclosing.GetEnclosingResults(ctx, v.ID, chunks)
	if err != nil {
		return ParticipantResults{}, err
	}
	scores := make(map[domain.ChunkID]float64, len(matches))
	for _, one := range matches {
		if _, has := scores[one.ChunkID]; !has {
			scores[one.ChunkID] = one.Score
		}
	}
	for i := range found {
		if score, has := scores[found[i].ChunkID]; has {
			found[i].Similarity = &score
		}
	}
	return ParticipantResults{Results: found}, nil
}
