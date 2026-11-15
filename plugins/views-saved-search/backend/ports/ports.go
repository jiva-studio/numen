// Package ports declares what the use cases of the saved search require of the host and of the reranking
// service.
package ports

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// Search searches the index of one vault and scores its chunks against conditions.
type Search interface {
	SearchText(ctx context.Context, query host.SearchResultQuery) ([]host.SearchResult, error)
	ScoreChunks(ctx context.Context, query host.ScoreQuery) (host.ChunkScores, error)
}

// NoteLookup reads the titles of the notes of one vault.
type NoteLookup interface {
	GetNotes(ctx context.Context, paths []string) (map[string]domain.NoteRef, error)
}

// Reranker scores documents against a query with a reranking model, one score for each document in the
// order of the documents. ErrNoReranker means no connection is set.
type Reranker interface {
	Rerank(ctx context.Context, query, instruction string, documents []string) ([]float64, error)
}

// ErrNoReranker is what a Reranker returns when the plugin has no connection for reranking.
var ErrNoReranker = errors.New("no connection for reranking is set")
