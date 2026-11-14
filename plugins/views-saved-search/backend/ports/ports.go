// Package ports declares what the use cases of the saved search require of the host and of the reranking
// service.
package ports

import (
	"context"
	"errors"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// Search searches the index of one vault and scores its chunks against conditions.
type Search interface {
	SearchText(ctx context.Context, query host.SearchResultQuery) ([]host.SearchResult, error)
	ScoreChunks(ctx context.Context, query host.ScoreQuery) (host.ChunkScores, error)
}

// NoteLookup reads the titles and identities of the notes of one vault.
type NoteLookup interface {
	GetNotes(ctx context.Context, paths []string) (map[string]domain.NoteRef, error)
}

// RerankerID is a row of the connections setting: its connection and the values of its row fields.
type RerankerID struct {
	ConnectionID string
	Options      map[string]string
}

// RerankRequest is the documents to score against a query. Floor is the similarity the run that kept the
// documents applied; a reranker that keeps scores reuses them only for the same floor.
type RerankRequest struct {
	Query       string
	Instruction string
	Documents   []string
	Priority    connection.Priority
	Floor       float64
}

// Reranked is one score for each document, in their order, and the row that gave them.
type Reranked struct {
	Scores   []float64
	Reranker RerankerID
}

// Reranker scores documents against a query, trying the rows of the setting in order. ErrNoReranker means
// no row is set.
type Reranker interface {
	// Rerank answers one score for each document, in their order, and the row that gave them.
	Rerank(ctx context.Context, request RerankRequest) (Reranked, error)
	// ListRerankers returns the rows that rerank, in the order they are tried.
	ListRerankers(ctx context.Context) ([]RerankerID, error)
}

// ErrNoReranker is what a Reranker returns when the plugin has no connection for reranking.
var ErrNoReranker = errors.New("no connection for reranking is set")

// Notes reads and writes the saved search notes of one vault.
type Notes interface {
	Read(ctx context.Context, path string) ([]byte, domain.FileStat, error)
	Write(ctx context.Context, path string, content []byte, baseline domain.FileStat) (domain.FileStat, error)
	Create(ctx context.Context, path string, content []byte) error
	Scan(ctx context.Context, paths []string) error
	GetPathsOfType(ctx context.Context, noteType domain.NoteType) ([]string, error)
}

// Store is the plugin's folder of one vault.
type Store interface {
	Read(ctx context.Context, name string) ([]byte, error)
	List(ctx context.Context) ([]host.StoreFile, error)
	Write(ctx context.Context, name string, content []byte) error
	Delete(ctx context.Context, name string) error
}

// Tasks reports the refresh to the host.
type Tasks interface {
	ReportTask(ctx context.Context, task host.Task) error
	FinishTask(ctx context.Context, id string) error
}

// Window names the vault the window shows.
type Window interface {
	GetVault(ctx context.Context) (domain.Vault, error)
}

// Clock is the time of the plugin.
type Clock interface {
	Now() time.Time
}
