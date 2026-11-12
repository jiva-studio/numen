// Package sourcefake is a search source for tests: it makes its vectors with an embedding function of the
// test and answers a query with the nearest chunks the index of the test finds.
package sourcefake

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
	sourcedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// DefaultFloor is the floor a query that names none applies, the default floor of the core.
const DefaultFloor = 0.30

// Embedder is the model of a test: its description and its vectors of texts.
type Embedder interface {
	Describe() searchsource.Description
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// Nearest finds the chunks of a vault nearest a vector, as the index of the core does.
type Nearest func(ctx context.Context, vaultID domain.VaultID, q sourcedomain.NearestQuery) ([]searchsource.Match, error)

// Source is a search source over the model of a test. A nil Nearest answers every query with nothing.
// Coverage is the text it says of every vault.
type Source struct {
	Model    Embedder
	Nearest  Nearest
	Floor    float64
	Coverage string
}

// Wrap returns the source over the model, which answers no query.
func Wrap(model Embedder) *Source {
	return &Source{Model: model}
}

// Over returns the source answering queries with the nearest chunks the function finds.
func (s *Source) Over(nearest Nearest) *Source {
	s.Nearest = nearest
	return s
}

// Describe returns the description of the model.
func (s *Source) Describe() searchsource.Description { return s.Model.Describe() }

// IndexTexts embeds the texts with the model.
func (s *Source) IndexTexts(ctx context.Context, _ searchsource.Place, texts []searchsource.Text) ([][]float32, error) {
	plain := make([]string, len(texts))
	for i, one := range texts {
		plain[i] = one.Text
	}
	return s.Model.Embed(ctx, plain)
}

// EmbedQueries embeds the queries with the model, with its queries where it tells them from documents.
func (s *Source) EmbedQueries(ctx context.Context, queries []string) ([][]float32, error) {
	if asking, ok := s.Model.(interface {
		EmbedQueries(ctx context.Context, queries []string) ([][]float32, error)
	}); ok {
		return asking.EmbedQueries(ctx, queries)
	}
	return s.Model.Embed(ctx, queries)
}

// SearchPassages embeds the query and returns the nearest chunks of the vault of the place.
func (s *Source) SearchPassages(ctx context.Context, place searchsource.Place, q searchsource.Query) ([]searchsource.Match, error) {
	if s.Nearest == nil {
		return nil, nil
	}
	vectors, err := s.EmbedQueries(ctx, []string{q.Text})
	if err != nil {
		return nil, err
	}
	if len(vectors) != 1 {
		return nil, errors.New("the model answered no vector for the query")
	}
	floor := q.Floor
	if floor <= 0 {
		floor = s.Floor
	}
	if floor <= 0 {
		floor = DefaultFloor
	}
	return s.Nearest(ctx, place.VaultID, sourcedomain.NearestQuery{
		Recipe: place.Recipe, Vector: vectors[0], Kinds: q.Kinds, PathPrefix: q.PathPrefix, Limit: q.Limit, Floor: floor,
	})
}

// DescribeCoverage returns the coverage text of the source.
func (s *Source) DescribeCoverage(context.Context, searchsource.Place, searchsource.Counts) (string, error) {
	return s.Coverage, nil
}

// Concurrency is the concurrency of the model where it names one, and one otherwise.
func (s *Source) Concurrency() int {
	if workers, ok := s.Model.(interface{ Concurrency() int }); ok && workers.Concurrency() > 0 {
		return workers.Concurrency()
	}
	return 1
}

// OnDescriptionChange registers fn with the model where it announces changes.
func (s *Source) OnDescriptionChange(fn func()) {
	if watcher, ok := s.Model.(interface{ OnModelChange(fn func()) }); ok {
		watcher.OnModelChange(fn)
	}
}
