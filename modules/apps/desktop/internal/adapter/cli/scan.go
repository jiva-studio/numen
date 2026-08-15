package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase"
)

func scanCommand(ctx context.Context, out io.Writer, cfg container.Config, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: numen scan <vault>")
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

	started := time.Now()
	res, err := usecase.ScanVault{Vaults: cfg.VaultReaders(), Notes: notes}.Execute(ctx, v)
	if err != nil {
		return err
	}
	stats, err := notes.Stats(ctx, v.ID)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "%s in %s\n", res, time.Since(started).Round(time.Millisecond))
	fmt.Fprintf(out, "index now holds %d notes, %d headings, %d tags\n",
		stats.Notes, stats.Headings, stats.Tags)
	return nil
}

// findVault resolves what the user typed and, when it resolves to nothing, says
// what to do about it — which is a matter of talking to a person, so it belongs
// here rather than in the use case.
func findVault(cfg container.Config, nameOrPath string) (domain.Vault, error) {
	registry, err := cfg.Registry()
	if err != nil {
		return domain.Vault{}, err
	}
	v, err := usecase.FindVault{Registry: registry}.Execute(nameOrPath)
	if err != nil {
		return domain.Vault{}, fmt.Errorf("%w — add it with: numen vault add %s", err, nameOrPath)
	}
	return v, nil
}
