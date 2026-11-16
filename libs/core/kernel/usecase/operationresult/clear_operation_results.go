package operationresult

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/port/operationport"
)

// ClearOperationResults removes every stored result of the connection operations.
type ClearOperationResults struct {
	cleaner operationport.OperationResultsCleaner
}

// NewClearOperationResults creates the use case over the stored results.
func NewClearOperationResults(cleaner operationport.OperationResultsCleaner) *ClearOperationResults {
	return &ClearOperationResults{cleaner: cleaner}
}

// Execute removes the stored results and returns how many it removed.
func (u *ClearOperationResults) Execute(ctx context.Context) (int, error) {
	return u.cleaner.ClearResults(ctx)
}
