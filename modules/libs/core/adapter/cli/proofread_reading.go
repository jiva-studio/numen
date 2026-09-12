package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// proofreadReadingCommand puts one document's reading right with a model.
//
// It reaches a service, so it runs where a person configured one. An
// installation that named none is told so and nothing is sent anywhere.
func proofreadReadingCommand(
	ctx context.Context,
	out io.Writer,
	deps Deps,
	v domain.Vault,
	path string,
) error {
	open, err := deps.ProofreadReading(ctx, v)
	if err != nil {
		return err
	}
	defer closing(open.Close)
	if !open.Held {
		return errors.New("nothing to proofread with: none is configured")
	}

	proofread, cut := open.Proofread, open.Cut
	fmt.Fprintf(out, "proofreading %s with %s\n", path, proofread.By.Name())
	started := time.Now()

	// The line of pages rewrites itself, and is closed once it stops.
	shown := false
	proofread.Cut = func(ctx context.Context, v domain.Vault, path string) error {
		_, err := cut.One(ctx, v, path)
		return err
	}
	proofread.OnProgress = func(res source.ProofreadReadingResult) {
		if res.Pages > 0 {
			fmt.Fprintf(out, "  page %d of %d\r", res.Read, res.Pages)
			shown = true
		}
	}
	res, err := proofread.Execute(ctx, v, path)
	if err != nil {
		return err
	}
	if shown {
		fmt.Fprintln(out)
	}

	switch {
	case res.Busy:
		fmt.Fprintf(out, "%s is already being proofread, and nothing was done\n", res.Path)
	case res.Waiting:
		// What a collected batch did is said here too: a run that leaves a
		// batch collects one before it.
		fmt.Fprintf(out, "put %d lines right, %d pages left as they were read; "+
			"%d of %d pages of %s are with the proofreader, ask again to collect them\n",
			res.Fixed, res.UncorrectedPages, res.Read, res.Pages, res.Path)
	case res.None:
		fmt.Fprintf(out, "%s has no reading to proofread\n", res.Path)
	default:
		fmt.Fprintf(out, "put %d lines of %s right over %d pages, %d of them left as they were read, in %s\n",
			res.Fixed, res.Path, res.Read, res.UncorrectedPages, time.Since(started).Round(time.Second))
	}
	return nil
}
