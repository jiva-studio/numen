package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/transcription"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// proofreadTranscriptCommand puts one recording's transcript right with a model.
//
// they are. A transcript a person edited is theirs, and is left as they left it.
func proofreadTranscriptCommand(
	ctx context.Context,
	out io.Writer,
	deps Deps,
	v domain.Vault,
	path string,
) error {
	open, err := deps.ProofreadTranscript(ctx, v)
	if err != nil {
		return err
	}
	defer closeIfOpen(open.Close)
	if !open.IsHeld {
		return errors.New("nothing to proofread with: none is configured")
	}

	proofread, cutSource := open.Proofread, source.NewCutSource(open.Cut)
	fmt.Fprintf(out, "proofreading %s with %s\n", path, proofread.GetProofreaderName())
	started := time.Now()

	// The line of lines rewrites itself, and is closed once it stops.
	isDisplayed := false
	proofread.Cut = cutSource.Execute
	proofread.OnProgress = func(res transcription.ProofreadResult) {
		if res.Lines > 0 {
			fmt.Fprintf(out, "  line %d of %d\r", res.Read, res.Lines)
			isDisplayed = true
		}
	}
	res, err := proofread.Execute(ctx, v, path)
	if err != nil {
		return err
	}
	if isDisplayed {
		fmt.Fprintln(out)
	}

	switch {
	case res.IsBusy:
		fmt.Fprintf(out, "%s is being listened to, and nothing was done\n", res.Path)
	case res.IsEdited:
		fmt.Fprintf(out, "the transcript of %s is somebody's own, and was left as it is\n", res.Path)
	case res.IsNone:
		fmt.Fprintf(out, "%s has no transcript to proofread\n", res.Path)
	default:
		fmt.Fprintf(out, "put %d lines of %s right over %d lines, %d batches left as transcribed, in %s\n",
			res.Fixed, res.Path, res.Read, res.UncorrectedBatches, time.Since(started).Round(time.Second))
	}
	return nil
}
