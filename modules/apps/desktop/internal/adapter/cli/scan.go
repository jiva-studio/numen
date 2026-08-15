package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/vault"
)

func scanCommand(ctx context.Context, out io.Writer, cfg container.Config, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: numen scan <vault>")
	}
	v, err := findVault(cfg, args[0])
	if err != nil {
		return err
	}
	db, err := cfg.Index(ctx)
	if err != nil {
		return err
	}
	defer db.Close()

	started := time.Now()
	result, err := vault.Scan{
		Readers: cfg.VaultReaders(),
		Vaults:  db.Vaults(),
		Notes:   db.Notes(),
		Known:   db.NoteQueries(),
	}.Execute(ctx, v)
	if err != nil {
		return err
	}
	summary, err := db.NoteQueries().Summary(ctx, v.ID)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "%s in %s\n", result, time.Since(started).Round(time.Millisecond))
	fmt.Fprintf(out, "index now holds %d notes and %d headings\n", summary.Notes, summary.Headings)
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
	v, err := vault.Find{Registry: registry}.Execute(nameOrPath)
	if err != nil {
		return domain.Vault{}, fmt.Errorf("%w — add it with: numen vault add %s", err, nameOrPath)
	}
	return v, nil
}
