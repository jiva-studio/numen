package cli

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vaultreading"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/note/domain"
)

type scanCommand struct {
	scan func(ctx context.Context, rebuild bool) (Scan, error)
}

func (scanCommand) Name() string { return "scan" }

func (scanCommand) Synopsis() string {
	return `  numen-cli scan <vault> [--rebuild-index]     update the index of a vault
                                               --rebuild-index reads every file again and deletes the vectors
                                               of every model except the one in use`
}

func (c scanCommand) Run(ctx context.Context, term terminal, args []string) (err error) {
	rebuild, rest := splitRebuildFlag(args)
	if len(rest) != 1 {
		return errors.New("usage: numen-cli scan <vault> [--rebuild-index]")
	}

	// The vectors are made here too. The window does all three in the
	// background; here they are waited for, which is this adapter's property
	// and not the use case's.
	open, err := c.scan(ctx, rebuild)
	if err != nil {
		return err
	}
	defer closeInto(&err, open.Close)
	r := newReporter(term.out)
	if open.Unembedded != nil {
		open.Read.Unembedded = open.Unembedded
	}
	showProgress(&open.Read, r)

	started := readClock(open.Now)
	v, made, err := open.Read.ExecuteByName(ctx, rest[0])
	if err != nil {
		return err
	}

	summary, err := open.Summary(ctx, v)
	if err != nil {
		return err
	}
	reportMade(r, v.Name, made)
	elapsed := ""
	if open.Now != nil {
		elapsed = ", in " + readClock(open.Now).Sub(started).Round(time.Millisecond).String()
	}
	r.Printf("index now holds %d notes and %d headings%s\n", summary.Notes, summary.Headings, elapsed)
	return r.Err()
}

func splitRebuildFlag(args []string) (rebuild bool, rest []string) {
	rest = make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--rebuild-index" {
			rebuild = true
			continue
		}
		rest = append(rest, arg)
	}
	return rebuild, rest
}

func readClock(now func() time.Time) time.Time {
	if now == nil {
		return time.Time{}
	}
	return now()
}

func describeSources(r vaultreading.IndexResult) string {
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

func showProgress(making *indexing.ReadWholeVault, r *reporter) {
	making.OnVaultFound = func(v domain.Vault, unembedded error) {
		if unembedded != nil {
			r.Printf("not embedding %s: %v\n", v.Name, unembedded)
		}
	}
	making.OnNotesProgress = func(res vaultreading.ScanResult) {
		r.Printf("  %d indexed\r", res.Indexed)
	}
	making.OnTextsProgress = func(res vaultreading.IndexResult) {
		if res.CurrentSource != "" {
			r.Printf("  cutting %s\r", res.CurrentSource)
		}
	}
	making.OnVectorsProgress = func(res vaultreading.EmbedResult) {
		r.Printf("  %d embedded\r", res.Embedded)
	}
}

func reportMade(r *reporter, vaultName string, made indexing.ReadWholeVaultResult) {
	if made.Texts.Seen > 0 {
		r.Println(describeSources(made.Texts))
	}
	switch {
	case made.Vectors.IsBusy:
		r.Printf("indexing %s is already running\n", vaultName)
	case made.Vectors.Embedded > 0:
		r.Printf("%d vectors made\n", made.Vectors.Embedded)
	}
}
