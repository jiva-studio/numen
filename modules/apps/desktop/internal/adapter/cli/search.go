package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase"
)

func searchCommand(ctx context.Context, out io.Writer, cfg container.Config, args []string) error {
	if len(args) < 2 {
		return errors.New("usage: numen search <vault> <query>")
	}
	v, err := findVault(cfg, args[0])
	if err != nil {
		return err
	}
	notes, err := cfg.Notes(ctx)
	if err != nil {
		return err
	}
	defer notes.Close()

	hits, err := usecase.SearchNotes{Notes: notes}.Execute(ctx, v, args[1])
	if err != nil {
		return err
	}
	if len(hits) == 0 {
		fmt.Fprintln(out, "nothing found")
		return nil
	}
	for _, h := range hits {
		fmt.Fprintf(out, "%s\n  %s\n  %s\n", h.Title, h.Path, h.Snippet)
	}
	return nil
}
