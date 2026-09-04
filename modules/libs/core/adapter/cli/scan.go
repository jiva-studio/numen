package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
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

	// The vectors are made here too. The window does all three in the
	// background; here they are waited for, which is this adapter's property
	// and not the use case's.
	embedder, closeEmbedder, why := cfg.Embedder(ctx)
	if why != nil {
		fmt.Fprintf(out, "not embedding %s: %v\n", v.Name, why)
	}
	if closeEmbedder != nil {
		defer func() { _ = closeEmbedder() }()
	}
	making, err := cfg.ReadWholeVault(ctx, db, embedder, v)
	if err != nil {
		return err
	}

	// A terminal that prints nothing for a minute looks broken. One group is
	// about half a second, and the line rewrites itself.
	making.Notes.OnProgress = func(res vaults.ScanResult) {
		fmt.Fprintf(out, "  %d indexed\r", res.Indexed)
	}
	making.Books.OnProgress = func(res source.ExtractResult) {
		if res.Reading != "" {
			fmt.Fprintf(out, "  reading %s\r", res.Reading)
		}
	}
	making.Vectors.OnProgress = func(res source.EmbedResult) {
		fmt.Fprintf(out, "  %d embedded\r", res.Embedded)
	}

	started := time.Now()
	made, err := making.Execute(ctx, v)
	if err != nil {
		return err
	}

	summary, err := db.Queries().Summary(ctx, v.ID)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "%s in %s\n", describe(made.Notes), time.Since(started).Round(time.Millisecond))
	if made.Books.Seen > 0 {
		fmt.Fprintln(out, describeSources(made.Books))
	}
	if made.Vectors.Embedded > 0 {
		fmt.Fprintf(out, "%d vectors made\n", made.Vectors.Embedded)
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
	v, err := vaults.Find{Registry: registry}.Execute(nameOrPath)
	if err != nil {
		return domain.Vault{}, fmt.Errorf("%w — add it with: numen-cli vault add %s", err, nameOrPath)
	}
	return v, nil
}

// describeSources puts the reading of what nobody typed here into words: the
// books of a vault and the recordings in it. Nothing is said about a vault
// holding none.
func describeSources(r source.ExtractResult) string {
	s := fmt.Sprintf("%d sources: %d read, %d unchanged, %d chunks",
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
func describe(r vaults.ScanResult) string {
	s := fmt.Sprintf("%d notes: %d indexed, %d unchanged, %d removed",
		r.Notes, r.Indexed, r.Unchanged, r.Removed)
	if r.Vanished > 0 {
		s += fmt.Sprintf(", %d gone before they could be read", r.Vanished)
	}
	return s
}
