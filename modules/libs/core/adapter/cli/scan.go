package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

func scanCommand(ctx context.Context, out io.Writer, deps Deps, args []string) error {
	// `--rebuild-index` reads every file, whatever the index remembers.
	rebuild := false
	rest := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--rebuild-index" {
			rebuild = true
			continue
		}
		rest = append(rest, arg)
	}
	if len(rest) != 1 {
		return errors.New("usage: numen-cli scan <vault> [--rebuild-index]")
	}
	v, err := findVault(deps, rest[0])
	if err != nil {
		return err
	}

	// The vectors are made here too. The window does all three in the
	// background; here they are waited for, which is this adapter's property
	// and not the use case's.
	open, err := deps.Scan(ctx, v, rebuild)
	if err != nil {
		return err
	}
	defer closeIfOpen(open.Close)
	if open.Unembedded != nil {
		fmt.Fprintf(out, "not embedding %s: %v\n", v.Name, open.Unembedded)
	}
	making := open.Read

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

	summary, err := open.Summary(ctx)
	if err != nil {
		return err
	}
	if made.Books.Seen > 0 {
		fmt.Fprintln(out, describeSources(made.Books))
	}
	if made.Vectors.Embedded > 0 {
		fmt.Fprintf(out, "%d vectors made\n", made.Vectors.Embedded)
	}
	fmt.Fprintf(out, "index now holds %d notes and %d headings, in %s\n",
		summary.Notes, summary.Headings, time.Since(started).Round(time.Millisecond))
	return nil
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
