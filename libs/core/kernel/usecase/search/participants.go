package search

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
	sourcedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
)

// Words ranks the chunks of a vault by the words of the query, through the full text index.
type Words struct {
	passages sourceport.PassageQueries
}

// NewWords returns the participant over the word queries of the index.
func NewWords(passages sourceport.PassageQueries) Words {
	return Words{passages: passages}
}

// Rank returns the chunks that hold the words of the query, best first.
func (w Words) Rank(ctx context.Context, v domain.Vault, q Query) (Ranking, error) {
	found, err := w.passages.FindLexical(ctx, v.ID, getLexicalQuery(q))
	return Ranking{Passages: found}, err
}

// Names ranks the sections of a vault whose names match the query. Its ranking leads: the section a
// source is found by comes before the other passages of that source.
type Names struct {
	passages sourceport.PassageQueries
}

// NewNames returns the participant over the section names of the index.
func NewNames(passages sourceport.PassageQueries) Names {
	return Names{passages: passages}
}

// Rank returns the sections whose names match the query, best first.
func (n Names) Rank(ctx context.Context, v domain.Vault, q Query) (Ranking, error) {
	found, err := n.passages.GetNamedPassages(ctx, v.ID, getLexicalQuery(q))
	return Ranking{Passages: found, IsLeading: true}, err
}

func getLexicalQuery(q Query) sourcedomain.LexicalQuery {
	return sourcedomain.LexicalQuery{Text: q.Text, Kinds: q.Kinds, Limit: q.Limit, IsGrowing: q.IsGrowing, PathPrefix: q.PathPrefix}
}

// SourceQueries is the search source a plugin provides, as a participant asks it.
type SourceQueries interface {
	// Describe is the description of the source, whose recipe names its place.
	Describe() searchsource.Description
	// SearchPassages returns the chunks of the place that answer the query, best first.
	SearchPassages(ctx context.Context, place searchsource.Place, q searchsource.Query) ([]searchsource.Match, error)
}

// EnclosingPassages reads the passages that enclose chunks of a vault.
type EnclosingPassages interface {
	// GetEnclosingPassages returns the passage that encloses each chunk of one vault, in the order
	// of `chunks`, leaving out a chunk the vault does not hold.
	GetEnclosingPassages(ctx context.Context, vaultID domain.VaultID, chunks []domain.ChunkID) ([]domain.Passage, error)
}

// Source is a search source of a plugin as a participant: it asks the source for the chunks of its place
// and turns them into the passages that enclose them.
type Source struct {
	source    SourceQueries
	enclosing EnclosingPassages
}

// NewSource returns the participant over a search source and the passages of the index.
func NewSource(source SourceQueries, enclosing EnclosingPassages) Source {
	return Source{source: source, enclosing: enclosing}
}

// Rank returns the passages enclosing the chunks the source answers, in the order of the source.
func (s Source) Rank(ctx context.Context, v domain.Vault, q Query) (Ranking, error) {
	place := searchsource.Place{VaultID: v.ID, Recipe: s.source.Describe().Recipe()}
	matches, err := s.source.SearchPassages(ctx, place, searchsource.Query{
		Text: q.Text, Kinds: q.Kinds, PathPrefix: q.PathPrefix, Limit: q.Limit, Floor: q.Floor,
	})
	if err != nil || len(matches) == 0 {
		return Ranking{}, err
	}
	chunks := make([]domain.ChunkID, len(matches))
	for i, one := range matches {
		chunks[i] = one.ChunkID
	}
	found, err := s.enclosing.GetEnclosingPassages(ctx, v.ID, chunks)
	return Ranking{Passages: found}, err
}
