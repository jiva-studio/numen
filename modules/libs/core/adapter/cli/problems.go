package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/jiva-studio/numen/modules/libs/core/check"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

func problemsCommand(ctx context.Context, out io.Writer, deps Deps, args []string) error {
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
	defer closing(open.Close)

	var named []domain.Check
	for _, name := range args[1:] {
		named = append(named, domain.Check(name))
	}

	found, err := check.Standard(open.Problems).Run(ctx, v, named...)
	if err != nil {
		return err
	}
	if len(found) == 0 {
		fmt.Fprintln(out, "nothing to report")
		return nil
	}
	for _, p := range found {
		fmt.Fprintf(out, "%s\n  %s: %s\n", p.Path, p.Kind, p.Detail)
	}
	return nil
}
