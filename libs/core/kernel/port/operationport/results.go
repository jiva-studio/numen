package operationport

import (
	"context"
	"time"
)

// OperationResults reads and writes the stored results of connection operations.
type OperationResults interface {
	// FindResults returns the stored outputs of the input hashes that have one, by input hash, and marks
	// them used at at.
	FindResults(ctx context.Context, operation, recipeHash string, inputHashes []string, at time.Time) (map[string][]byte, error)
	// SaveResults stores the outputs by input hash under the operation and recipe, used at at, replacing
	// equal keys, then removes the least recently used rows above the row limit of the store.
	SaveResults(ctx context.Context, operation, recipeHash string, outputs map[string][]byte, at time.Time) error
}

// OperationResultsCleaner removes stored results.
type OperationResultsCleaner interface {
	// ClearResults removes every stored result and returns how many it removed.
	ClearResults(ctx context.Context) (int, error)
}
