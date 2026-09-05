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

// proofreadTranscriptCommand puts one recording's transcript right with a model.
//
// The words are corrected and the moments they were spoken at are left where
// they are. A transcript a person edited is theirs, and is left as they left it.
func proofreadTranscriptCommand(
	ctx context.Context,
	out io.Writer,
	cfg container.Config,
	v domain.Vault,
	path string,
) error {
	proofread, held, err := cfg.ProofreadingSpeech().
		Transcript(cfg.VaultReaders(), cfg.DerivedStores())
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

	// What a batch puts right is cut before the next is asked about, so a
	// recording answers about the speech already corrected while the rest is
	// still being asked about.
	cut, err := cfg.Extract(db.Sources(), db.SourcesKnown(), v)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "proofreading %s with %s\n", path, proofread.By.Name())
	started := time.Now()

	// The line of lines rewrites itself, and is closed once it stops.
	shown := false
	proofread.Cut = func(ctx context.Context, v domain.Vault, path string) error {
		_, err := cut.One(ctx, v, path)
		return err
	}
	proofread.OnProgress = func(res source.ProofreadTranscriptResult) {
		if res.Lines > 0 {
			fmt.Fprintf(out, "  line %d of %d\r", res.Read, res.Lines)
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
		fmt.Fprintf(out, "%s is being listened to, and nothing was done\n", res.Path)
	case res.Edited:
		fmt.Fprintf(out, "the transcript of %s is somebody's own, and was left as it is\n", res.Path)
	case res.None:
		fmt.Fprintf(out, "%s has no transcript to proofread\n", res.Path)
	default:
		fmt.Fprintf(out, "put %d lines of %s right over %d lines, %d batches left as they were heard, in %s\n",
			res.Fixed, res.Path, res.Read, res.Refused, time.Since(started).Round(time.Second))
	}
	return nil
}
