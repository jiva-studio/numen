package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
)

func problemsCommand(ctx context.Context, out io.Writer, cfg container.Config, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: numen problems <vault>")
	}
	v, err := findVault(cfg, args[0])
	if err != nil {
		return err
	}
	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		return err
	}
	defer db.Close()

	found, err := note.ListProblems{Problems: db.Problems()}.Execute(ctx, v)
	if err != nil {
		return err
	}
	if len(found) == 0 {
		fmt.Fprintln(out, "nothing to report")
		return nil
	}
	for _, p := range found {
		fmt.Fprintf(out, "%s\n  %s\n", p.Path, p.Detail)
	}
	return nil
}
