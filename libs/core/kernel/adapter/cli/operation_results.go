package cli

import (
	"context"
	"errors"
)

const operationResultsUsage = "usage: numen-cli operation-results clear"

type operationResultsCommand struct {
	session Session
}

func (operationResultsCommand) Name() string { return "operation-results" }

func (operationResultsCommand) Synopsis() string {
	return "  numen-cli operation-results clear            remove the stored results of connection operations"
}

func (c operationResultsCommand) Run(ctx context.Context, term terminal, args []string) error {
	if len(args) != 1 || args[0] != "clear" {
		return errors.New(operationResultsUsage)
	}
	indexed, err := c.session.OpenIndexed(ctx, IndexOptions{})
	if err != nil {
		return err
	}
	removed, err := indexed.OperationResults.Execute(ctx)
	if err != nil {
		return err
	}
	r := newReporter(term.out)
	r.WriteFormat("removed %d stored results\n", removed)
	return r.Err()
}
