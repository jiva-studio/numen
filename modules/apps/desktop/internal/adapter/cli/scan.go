package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	usecase "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/vault"
)

func scanCommand(ctx context.Context, out io.Writer, cfg container.Config, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: numen-cli scan <vault>")
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

	started := time.Now()
	scan := usecase.Scan{
		Readers:     cfg.VaultReaders(),
		Vaults:      db.Vaults(),
		Notes:       db.Notes(),
		Known:       db.Queries(),
		Maintenance: db.Maintenance(),
	}
	// A terminal that prints nothing for a minute looks broken. One group is
	// about half a second, and the line rewrites itself.
	scan.OnProgress = func(res usecase.ScanResult) {
		fmt.Fprintf(out, "  %d indexed\r", res.Indexed)
	}
	result, err := scan.Execute(ctx, v)
	if err != nil {
		return err
	}
	summary, err := db.Queries().Summary(ctx, v.ID)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "%s in %s\n", describe(result), time.Since(started).Round(time.Millisecond))
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
	v, err := usecase.Find{Registry: registry}.Execute(nameOrPath)
	if err != nil {
		return domain.Vault{}, fmt.Errorf("%w — add it with: numen-cli vault add %s", err, nameOrPath)
	}
	return v, nil
}

// describe puts a scan into words. The use case counts; how that is said to a
// person belongs to this adapter, and a graphical shell will say it differently
// or not at all.
func describe(r usecase.ScanResult) string {
	s := fmt.Sprintf("%d notes: %d indexed, %d unchanged, %d removed",
		r.Seen, r.Indexed, r.Unchanged, r.Removed)
	if r.Vanished > 0 {
		s += fmt.Sprintf(", %d gone before they could be read", r.Vanished)
	}
	return s
}
