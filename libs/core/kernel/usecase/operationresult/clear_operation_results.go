package operationresult

import (
	"context"
)

// ClearOperationResults removes every stored result of the connection operations.
type ClearOperationResults struct {
	cleaner OperationResultsCleaner
}

// NewClearOperationResults creates the use case over the stored results.
func NewClearOperationResults(cleaner OperationResultsCleaner) *ClearOperationResults {
	return &ClearOperationResults{cleaner: cleaner}
}

// Execute removes the stored results and returns how many it removed.
func (u *ClearOperationResults) Execute(ctx context.Context) (int, error) {
	return u.cleaner.ClearResults(ctx)
}
