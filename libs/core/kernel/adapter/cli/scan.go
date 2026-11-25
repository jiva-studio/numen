package cli

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vaultreading"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

type scanCommand struct {
	session Session
}

func (scanCommand) Name() string { return "scan" }

func (scanCommand) Synopsis() string {
	return `  numen-cli scan <vault> [--rebuild-index]     update the index of a vault
                                               --rebuild-index reads every file again and deletes the vectors
                                               of every model except the one in use`
}

func (c scanCommand) Run(ctx context.Context, term terminal, args []string) error {
	rebuild, rest := splitRebuildFlag(args)
	if len(rest) != 1 {
		return errors.New("usage: numen-cli scan <vault> [--rebuild-index]")
	}

	// The vectors are made here too. The window does all three in the
	// background; here they are waited for, which is this adapter's property
	// and not the use case's.
	indexed, err := c.session.OpenIndexed(ctx, IndexOptions{Rebuild: rebuild})
	if err != nil {
		return err
	}
	open := indexed.Scan
	r := newReporter(term.out)
	started := readClock(open.Now)
	scanned, err := open.Scan.Execute(ctx, rest[0], getProgressReporting(r))
	if err != nil {
		return err
	}

	summary := scanned.Summary
	reportMade(r, scanned.Vault.Name, scanned.Made)
	elapsed := ""
	if open.Now != nil {
		elapsed = ", in " + readClock(open.Now).Sub(started).Round(time.Millisecond).String()
	}
	r.WriteFormat("index now holds %d notes and %d headings%s\n", summary.Notes, summary.Headings, elapsed)
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

func getProgressReporting(r *reporter) indexing.ReadCallbacks {
	return indexing.ReadCallbacks{
		NotifyVaultFound: func(v domain.Vault, unembedded error) {
			if unembedded != nil {
				r.WriteFormat("not embedding %s: %v\n", v.Name, unembedded)
			}
		},
		NotifyNotesProgress: func(res vaultreading.ScanResult) {
			r.WriteFormat("  %d indexed\r", res.Indexed)
		},
		NotifyTextsProgress: func(res vaultreading.IndexResult) {
			if res.CurrentSource != "" {
				r.WriteFormat("  cutting %s\r", res.CurrentSource)
			}
		},
		NotifyVectorsProgress: func(res vaultreading.EmbedResult) {
			r.WriteFormat("  %d embedded\r", res.Embedded)
		},
	}
}

func reportMade(r *reporter, vaultName string, made indexing.ReadWholeVaultResult) {
	if made.Texts.Seen > 0 {
		r.WriteLine(describeSources(made.Texts))
	}
	switch {
	case made.Vectors.IsBusy:
		r.WriteFormat("indexing %s is already running\n", vaultName)
	case made.Vectors.Embedded > 0:
		r.WriteFormat("%d vectors made\n", made.Vectors.Embedded)
	}
}
