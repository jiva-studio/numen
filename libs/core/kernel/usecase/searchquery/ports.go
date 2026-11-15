package searchquery

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// QueryParser parses the text of a query against a catalog. A query that does not parse or type-check
// returns *searchquery.ParseError with every problem; another error is a failure of the parser.
type QueryParser interface {
	ParseQuery(text string, catalog searchquery.Catalog) (searchquery.RawNode, error)
}

// DocumentQueries lists the documents of a vault within a scope superset.
type DocumentQueries interface {
	// ListDocuments returns the sources of one vault under any of q.PathPrefixes and of q.SourceKinds, with
	// their note title, size, time of change and the count of their small chunks that hold a vector under
	// q.Recipe, ordered by path, at most q.Limit+1 of them, all of them when q.Limit is zero.
	ListDocuments(ctx context.Context, vaultID domain.VaultID, q searchquery.DocumentQuery) ([]searchquery.Document, error)
}

// VectorQueries reads vectors for a search by meaning.
type VectorQueries interface {
	// FindCoarseCandidates returns the chunks whose one-bit vectors are nearest the query, nearest first.
	FindCoarseCandidates(ctx context.Context, vaultID domain.VaultID, q source.CoarseQuery) ([]domain.ChunkID, error)
	// ListDocumentVectors returns the vectors under q.Recipe of the small chunks of the documents at q.Paths,
	// at most q.Limit after chunk q.After, ordered by chunk id.
	ListDocumentVectors(ctx context.Context, vaultID domain.VaultID, q searchquery.DocumentVectorQuery) ([]source.StoredVector, error)
	// ListStoredVectors returns the full-precision vectors of the chunks under q.Recipe.
	ListStoredVectors(ctx context.Context, q source.StoredVectorQuery) ([]source.StoredVector, error)
}

// WordQueries reads the word index for a words condition.
type WordQueries interface {
	// FindWords returns the small chunks whose text matches the FTS5 expression, with their negated bm25,
	// best first, within q.Paths when set, else within q.PathPrefixes and q.SourceKinds, at most q.Limit. An
	// expression FTS5 cannot parse returns source.ErrInvalidWords.
	FindWords(ctx context.Context, vaultID domain.VaultID, q searchquery.WordQuery) ([]searchquery.WordMatch, error)
	// GetWordScores returns the negated bm25 of each given chunk that matches the expression.
	GetWordScores(ctx context.Context, vaultID domain.VaultID, expression string, chunks []domain.ChunkID) (map[domain.ChunkID]float64, error)
}

// FloorReader returns the similarity floor of a vault under a recipe.
type FloorReader interface {
	GetFloor(ctx context.Context, v domain.Vault, recipe string) float64
}
