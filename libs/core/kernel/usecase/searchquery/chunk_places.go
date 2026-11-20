package searchquery

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// ChunkPlaces reads the large chunk that encloses a chunk, the small chunks inside it and their vectors.
type ChunkPlaces interface {
	// GetEnclosingResults returns the search result that encloses each chunk of one vault, in the order
	// of `chunks`, leaving out a chunk the vault does not hold.
	GetEnclosingResults(ctx context.Context, vaultID domain.VaultID, chunks []domain.ChunkID) ([]domain.SearchResult, error)
	// ListSmallChunks returns the small chunks inside the large chunk that encloses each chunk of one vault.
	ListSmallChunks(ctx context.Context, vaultID domain.VaultID, chunks []domain.ChunkID) ([]source.SmallChunk, error)
	// ListStoredVectors returns the vectors kept under `q.Recipe` for the chunks of `q.Chunks`, leaving
	// out the chunks without one.
	ListStoredVectors(ctx context.Context, q source.StoredVectorQuery) ([]source.StoredVector, error)
}
