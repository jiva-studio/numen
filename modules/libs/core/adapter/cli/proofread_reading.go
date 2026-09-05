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
)

// proofreadCommand puts one file right with a model, by what the file is: a
// recording's transcript, or a document's reading.
func proofreadCommand(ctx context.Context, out io.Writer, cfg container.Config, deps Deps, args []string) error {
	if len(args) != 2 {
		return errors.New("usage: numen-cli proofread <vault> <file>")
	}
	v, err := findVault(deps, args[0])
	if err != nil {
		return err
	}
	if domain.MediaType(args[1]) != "" {
		return proofreadTranscriptCommand(ctx, out, cfg, v, args[1])
	}
	return proofreadReadingCommand(ctx, out, cfg, v, args[1])
}

// proofreadReadingCommand puts one document's reading right with a model.
//
// It reaches a service, so it runs where a person configured one. An
// installation that named none is told so and nothing is sent anywhere.
func proofreadReadingCommand(
	ctx context.Context,
	out io.Writer,
	cfg container.Config,
	v domain.Vault,
	path string,
) error {
	proofread, held, err := cfg.ProofreadingScans().Reading(cfg.VaultReaders(), cfg.DerivedStores())
	if err != nil {
		return err
	}
	if !held {
		return errors.New("nothing to proofread with: none is configured")
	}
	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		return err
	}
	defer db.Close()

	// What a batch of pages puts right is cut before the next batch is asked
	// about, so a book answers about the pages already corrected while the rest
	// is still being asked about.
	cut, err := cfg.Extract(db.Sources(), db.SourcesKnown(), v)
	if err != nil {
		return err
	}

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
			res.Fixed, res.Refused, res.Read, res.Pages, res.Path)
	case res.None:
		fmt.Fprintf(out, "%s has no reading to proofread\n", res.Path)
	default:
		fmt.Fprintf(out, "put %d lines of %s right over %d pages, %d of them left as they were read, in %s\n",
			res.Fixed, res.Path, res.Read, res.Refused, time.Since(started).Round(time.Second))
	}
	return nil
}
