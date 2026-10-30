package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/jiva-studio/numen/libs/core/usecase/filetext"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/usecase/vectors"
)

func runScanCommand(ctx context.Context, out io.Writer, deps Deps, args []string) (err error) {
	rebuild, rest := splitRebuildFlag(args)
	if len(rest) != 1 {
		return errors.New("usage: numen-cli scan <vault> [--rebuild-index]")
	}
	registered, err := deps.Vaults()
	if err != nil {
		return err
	}
	v, err := registered.find(rest[0])
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
	defer closeInto(&err, open.Close)
	r := newReporter(out)
	if open.Unembedded != nil {
		r.Printf("not embedding %s: %v\n", v.Name, open.Unembedded)
	}
	making := open.Read

	// A terminal that prints nothing for a minute looks broken. One group is
	// about half a second, and the line rewrites itself.
	making.OnNotesProgress = func(res vaults.ScanResult) {
		r.Printf("  %d indexed\r", res.Indexed)
	}
	making.OnTextsProgress = func(res filetext.IndexResult) {
		if res.CurrentSource != "" {
			r.Printf("  cutting %s\r", res.CurrentSource)
		}
	}
	making.OnVectorsProgress = func(res vectors.EmbedResult) {
		r.Printf("  %d embedded\r", res.Embedded)
	}

	started := readClock(registered.Now)
	made, err := making.Execute(ctx, v)
	if err != nil {
		return err
	}

	summary, err := open.Summary(ctx)
	if err != nil {
		return err
	}
	if made.Texts.Seen > 0 {
		r.Println(describeSources(made.Texts))
	}
	switch {
	case made.Vectors.IsBusy:
		r.Printf("indexing %s is already running\n", v.Name)
	case made.Vectors.Embedded > 0:
		r.Printf("%d vectors made\n", made.Vectors.Embedded)
	}
	elapsed := ""
	if registered.Now != nil {
		elapsed = ", in " + readClock(registered.Now).Sub(started).Round(time.Millisecond).String()
	}
	r.Printf("index now holds %d notes and %d headings%s\n", summary.Notes, summary.Headings, elapsed)
	return r.Err()
}

// splitRebuildFlag separates `--rebuild-index`, which reads every file whatever
// the index remembers, from the other arguments.
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

// readClock is the time the clock gives, and the zero time where there is no clock.
func readClock(clock func() time.Time) time.Time {
	if clock == nil {
		return time.Time{}
	}
	return clock()
}

// describeSources describes the extraction of the files that are not notes: the
// files the plugins read text from. Nothing is reported for a vault holding none.
func describeSources(r filetext.IndexResult) string {
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
