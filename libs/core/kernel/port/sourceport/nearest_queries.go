package sourceport

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// NearestQueries is what the index returns for a search by meaning, unranked: the candidates of the
// coarse pass, the vectors of some chunks and the search result enclosing a chunk. The caller orders the
// candidates, applies the similarity floor and decides how many to keep.
type NearestQueries interface {
	// FindCoarseCandidates returns the chunks of one vault nearest `q.Vector` by one bit per
	// dimension, nearest first, at most `q.Limit` of them. It fails with `source.ErrOtherRecipe`
	// where the index keeps its vectors under another recipe than `q.Recipe`.
	FindCoarseCandidates(ctx context.Context, vaultID domain.VaultID, q source.CoarseQuery) ([]domain.ChunkID, error)

	// ListStoredVectors returns the vectors kept under `q.Recipe` for the chunks of `q.Chunks`,
	// leaving out the chunks without one and those of other kinds than `q.Kinds`.
	ListStoredVectors(ctx context.Context, q source.StoredVectorQuery) ([]source.StoredVector, error)

	// GetEnclosingResults returns the search result that encloses each chunk of one vault, in the order
	// of `chunks`, leaving out a chunk the vault does not hold.
	GetEnclosingResults(ctx context.Context, vaultID domain.VaultID, chunks []domain.ChunkID) ([]domain.SearchResult, error)
}
