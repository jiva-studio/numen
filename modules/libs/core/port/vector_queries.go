package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// VectorQueries answers what the vector index does not hold. Progress is read
// from the data: what owes a vector is a chunk that has none from the model in
// use.
type VectorQueries interface {
	// Unembedded is the small chunks of one vault with no vector from the model
	// given, in ascending order of chunk, from `after` onwards. Asked again with
	// the last chunk of the previous answer, it carries on where it stopped.
	//
	// A large chunk carries no vector and is never in the answer. Neither is
	// the text: a chunk is a place in a file, and the file holds the words.
	Unembedded(ctx context.Context, vaultID domain.VaultID, model EmbeddingModel, after int64, limit int) ([]domain.Passage, error)
}
