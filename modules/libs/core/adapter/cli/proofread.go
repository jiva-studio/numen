package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// proofreadCommand puts one document's reading right with a model.
//
// It reaches a service, so it runs where a person configured one. An
// installation that named none is told so and nothing is sent anywhere.
func proofreadCommand(ctx context.Context, out io.Writer, cfg container.Config, args []string) error {
	if len(args) != 2 {
		return errors.New("usage: numen-cli proofread <vault> <file>")
	}
	v, err := findVault(cfg, args[0])
	if err != nil {
		return err
	}
	if domain.MediaType(args[1]) != "" {
		return putRightCommand(ctx, out, cfg, v, args[1])
	}
	named := cfg.ScanProofreading.Profile
	by, err := cfg.Proofreader(named, proofread.ScanInstruction)
	if err != nil {
		return fmt.Errorf("nothing to proofread with: %w", err)
	}
	if by == nil {
		return errors.New("nothing to proofread with: none is configured")
	}
	queue, err := cfg.ProofreadQueue(named, proofread.ScanInstruction)
	if err != nil {
		return fmt.Errorf("nothing to leave the pages with: %w", err)
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

	fmt.Fprintf(out, "proofreading %s with %s\n", args[1], by.Name())
	started := time.Now()

	// The line of pages rewrites itself, and is closed once it stops.
	shown := false
	res, err := source.Proofread{
		Readers:         cfg.VaultReaders(),
		Derived:         cfg.DerivedStores(),
		By:              by,
		Queue:           queue,
		Pages:           cfg.Proofreading.Profiles[named].BatchSize,
		MaxEditDistance: cfg.Proofreading.Distance(),
		Cut: func(ctx context.Context, v domain.Vault, path string) error {
			_, err := cut.One(ctx, v, path)
			return err
		},
		OnProgress: func(res source.ProofreadResult) {
			if res.Pages > 0 {
				fmt.Fprintf(out, "  page %d of %d\r", res.Read, res.Pages)
				shown = true
			}
		},
	}.Execute(ctx, v, args[1])
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
