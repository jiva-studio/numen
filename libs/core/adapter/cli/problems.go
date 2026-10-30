package cli

import (
	"context"
	"errors"
	"io"

	"github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

func runProblemsCommand(ctx context.Context, out io.Writer, deps Deps, args []string) (err error) {
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
	defer closeInto(&err, open.Close)

	var named []domain.Check
	for _, name := range args[1:] {
		named = append(named, domain.Check(name))
	}

	report, err := vault.NewStandardGetProblems(open.Problems).Execute(ctx, v, named...)
	if err != nil {
		return err
	}
	r := newReporter(out)
	if len(report.Found) == 0 {
		r.Println("nothing to report")
		return r.Err()
	}
	for _, p := range report.Found {
		r.Printf("%s\n  %s: %s\n", p.Path, p.Kind, p.Detail)
	}
	return r.Err()
}
