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

// putRightCommand puts one recording's transcript right with a model.
//
// The words are corrected and the moments they were spoken at are left where
// they are. A transcript a person edited is theirs, and is left as they left it.
func putRightCommand(
	ctx context.Context,
	out io.Writer,
	cfg container.Config,
	v domain.Vault,
	path string,
) error {
	named := cfg.SpeechProofreading.With
	by, err := cfg.Proofreader(named, proofread.SpeechInstruction)
	if err != nil {
		return fmt.Errorf("nothing to proofread with: %w", err)
	}
	if by == nil {
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

	fmt.Fprintf(out, "proofreading %s with %s\n", path, by.Name())
	started := time.Now()

	// The line of lines rewrites itself, and is closed once it stops.
	shown := false
	profile := cfg.Proofreading.Profiles[named]
	res, err := source.PutRight{
		Readers: cfg.VaultReaders(),
		Derived: cfg.DerivedStores(),
		By:      by,
		Lines:   profile.BatchSize,
		Overlap: profile.Overlap,
		Apart:   cfg.Proofreading.Apart(),
		Cut: func(ctx context.Context, v domain.Vault, path string) error {
			_, err := cut.One(ctx, v, path)
			return err
		},
		OnProgress: func(res source.PutRightResult) {
			if res.Lines > 0 {
				fmt.Fprintf(out, "  line %d of %d\r", res.Read, res.Lines)
				shown = true
			}
		},
	}.Execute(ctx, v, path)
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
		fmt.Fprintf(out, "put %d lines of %s right over %d, %d of them left as they were heard, in %s\n",
			res.Fixed, res.Path, res.Read, res.Refused, time.Since(started).Round(time.Second))
	}
	return nil
}
