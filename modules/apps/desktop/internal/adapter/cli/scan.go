package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/source"
	usecase "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/vault"
)

func scanCommand(ctx context.Context, out io.Writer, cfg container.Config, args []string) error {
	// `--rebuild-index` reads every file, whatever the index remembers.
	rest := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--rebuild-index" {
			cfg.RebuildIndex = true
			continue
		}
		rest = append(rest, arg)
	}
	if len(rest) != 1 {
		return errors.New("usage: numen-cli scan <vault> [--rebuild-index]")
	}
	v, err := findVault(cfg, rest[0])
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
		Readers:      cfg.VaultReaders(),
		Vaults:       db.Vaults(),
		Notes:        db.Notes(),
		Known:        db.Queries(),
		Maintenance:  db.Maintenance(),
		RebuildIndex: cfg.RebuildIndex,
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
	// A scan reads the whole vault, and a book in it is part of the vault. The
	// window does the same in the background; here it is waited for, which is
	// this adapter's property and not the use case's.
	derived, err := cfg.DerivedStores().Open(v)
	if err != nil {
		return err
	}
	extract := source.Extract{
		Readers:      cfg.VaultReaders(),
		Sources:      db.Sources(),
		Owing:        db.SourcesKnown(),
		Derived:      derived,
		RebuildIndex: cfg.RebuildIndex,
		OnProgress: func(res source.ExtractResult) {
			if res.Reading != "" {
				fmt.Fprintf(out, "  reading %s\r", res.Reading)
			}
		},
	}
	sources, err := extract.Execute(ctx, v)
	if err != nil {
		return err
	}

	summary, err := db.Queries().Summary(ctx, v.ID)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "%s in %s\n", describe(result), time.Since(started).Round(time.Millisecond))
	if sources.Seen > 0 {
		fmt.Fprintln(out, describeSources(sources))
	}
	fmt.Fprintf(out, "index now holds %d notes and %d headings\n", summary.Notes, summary.Headings)
	return nil
}

// findVault resolves what the person typed and, when it resolves to nothing,
// says what to do about it. Talking to a person belongs here.
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

// describeSources puts the reading of books into words. Nothing is said about a
// vault holding none.
func describeSources(r source.ExtractResult) string {
	s := fmt.Sprintf("%d books: %d read, %d unchanged, %d chunks",
		r.Seen, r.Extracted, r.Unchanged, r.Chunks)
	if r.Unreadable > 0 {
		s += fmt.Sprintf(", %d could not be read", r.Unreadable)
	}
	if r.Vanished > 0 {
		s += fmt.Sprintf(", %d gone before they could be read", r.Vanished)
	}
	return s
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
