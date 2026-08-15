package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
)

func searchCommand(ctx context.Context, out io.Writer, cfg container.Config, args []string) error {
	if len(args) < 2 {
		return errors.New("usage: numen search <vault> <query>")
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

	matches, err := note.Search{Notes: db.Queries()}.Execute(ctx, v, args[1])
	if err != nil {
		return err
	}
	if len(matches) == 0 {
		fmt.Fprintln(out, "nothing found")
		return nil
	}
	for _, m := range matches {
		fmt.Fprintf(out, "%s\n  %s\n", m.Title, m.Path)
	}
	return nil
}
