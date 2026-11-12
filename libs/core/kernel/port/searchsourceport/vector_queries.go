package searchsourceport

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vector"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// VectorQueries answers what the vector index does not hold. Progress is
// derived from the data: a chunk that needs a vector is one that has none under
// the recipe in use.
type VectorQueries interface {
	// GetUnembeddedChunks returns at most `limit` small chunks of one vault with
	// no vector under the given recipe, starting at `after`, and the cursor for the
	// next call. Calling again with that cursor continues where the previous call stopped.
	//
	// A large chunk carries no vector and is never in the answer. Neither is
	// the text: a chunk is a place in a file, and the file holds the words.
	GetUnembeddedChunks(ctx context.Context, vaultID domain.VaultID, recipe string, after vector.ChunkCursor, limit int) ([]domain.Passage, vector.ChunkCursor, error)
}
