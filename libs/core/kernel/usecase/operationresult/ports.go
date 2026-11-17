package operationresult

import "context"

// OperationResultsCleaner removes stored results.
type OperationResultsCleaner interface {
	// ClearResults removes every stored result and returns how many it removed.
	ClearResults(ctx context.Context) (int, error)
}
