package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

func runProblemsCommand(ctx context.Context, out io.Writer, deps Deps, args []string) error {
	if len(args) < 1 {
		return errors.New("usage: numen-cli problems <vault> [<check>...]")
	}
	v, err := findVault(deps, args[0])
	if err != nil {
		return err
	}
	open, err := deps.Problems(ctx)
	if err != nil {
		return err
	}
	defer closeIfOpen(open.Close)

	var named []domain.Check
	for _, name := range args[1:] {
		named = append(named, domain.Check(name))
	}

	report, err := vault.NewStandardGetProblems(open.Problems).Execute(ctx, v, named...)
	if err != nil {
		return err
	}
	if len(report.Found) == 0 {
		fmt.Fprintln(out, "nothing to report")
		return nil
	}
	for _, p := range report.Found {
		fmt.Fprintf(out, "%s\n  %s: %s\n", p.Path, p.Kind, p.Detail)
	}
	return nil
}
