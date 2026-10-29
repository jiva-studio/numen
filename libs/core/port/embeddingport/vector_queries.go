package embeddingport

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// ChunkCursor is a place in a walk over the chunks of one vault. The index
// writes it and reads it back, and the empty cursor is the beginning: a caller
// carries it from one answer to the next and reads nothing out of it.
type ChunkCursor string

// VectorQueries answers what the vector index does not hold. Progress is
// derived from the data: a chunk that needs a vector is one that has none from
// the model in use.
type VectorQueries interface {
	// GetUnembeddedChunks returns at most `limit` small chunks of one vault with
	// no vector from the given model, starting at `after`, and the cursor for the
	// next call. Calling again with that cursor continues where the previous call stopped.
	//
	// A large chunk carries no vector and is never in the answer. Neither is
	// the text: a chunk is a place in a file, and the file holds the words.
	GetUnembeddedChunks(ctx context.Context, vaultID domain.VaultID, model EmbeddingModel, after ChunkCursor, limit int) ([]domain.Passage, ChunkCursor, error)
}
