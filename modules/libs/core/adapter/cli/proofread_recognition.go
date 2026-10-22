package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/recognition"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// proofreadRecognitionCommand puts one document's recognition right with a model.
//
// It reaches a service, so it runs where a person configured one. An
// installation that named none is told so and nothing is sent anywhere.
func proofreadRecognitionCommand(
	ctx context.Context,
	out io.Writer,
	deps Deps,
	v domain.Vault,
	path string,
) error {
	open, err := deps.ProofreadRecognition(ctx, v)
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

	// The line of pages rewrites itself, and is closed once it stops.
	isDisplayed := false
	proofread.Cut = cutSource.Execute
	proofread.OnProgress = func(res recognition.ProofreadResult) {
		if res.Pages > 0 {
			fmt.Fprintf(out, "  page %d of %d\r", res.Read, res.Pages)
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
		fmt.Fprintf(out, "%s is already being proofread, and nothing was done\n", res.Path)
	case res.IsNone:
		fmt.Fprintf(out, "%s has no recognition to proofread\n", res.Path)
	default:
		fmt.Fprintf(out, "put %d lines of %s right over %d pages, %d of them left as they were read, in %s\n",
			res.Fixed, res.Path, res.Read, res.UncorrectedPages, time.Since(started).Round(time.Second))
	}
	return nil
}
