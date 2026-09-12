package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/transcript"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// transcribeCommand listens to one recording with a model and writes down what
// it heard.
//
// A recording carries no text of its own, so what the model heard is the only
// text there is. The window listens to a vault's recordings on its own; this is
// the hand asking for one.
func transcribeCommand(ctx context.Context, out io.Writer, deps Deps, args []string) error {
	again := false
	rest := make([]string, 0, len(args))
	for _, one := range args {
		if one == "--again" {
			again = true
			continue
		}
		rest = append(rest, one)
	}
	args = rest
	if len(args) != 2 {
		return errors.New("usage: numen-cli transcribe <vault> <file> [--again]")
	}
	v, err := findVault(deps, args[0])
	if err != nil {
		return err
	}
	// A terminal is where waiting is what a person came for, so the opener
	// fetches what is missing and waits for it. It is said out loud first: a
	// program that prints nothing for minutes looks broken.
	fetching := func() {
		fmt.Fprintln(out, "fetching what is needed to transcribe recordings")
	}
	open, err := deps.Transcribe(ctx, v, fetching)
	if err != nil {
		return fmt.Errorf("nothing to transcribe with: %w", err)
	}
	defer closing(open.Close)

	transcribe, cut := open.Transcribe, open.Cut
	fmt.Fprintf(out, "transcribing %s with %s\n", args[1], transcribe.By.Transcription())
	started := time.Now()

	// The line of minutes is closed once it stops, so what follows it stands on
	// a line of its own.
	shown := false
	transcribe.Again = again
	transcribe.Cut = func(ctx context.Context, v domain.Vault, path string) error {
		_, err := cut.One(ctx, v, path)
		return err
	}
	// A terminal that prints nothing for an hour looks broken, and this takes
	// about that. The line rewrites itself.
	transcribe.OnProgress = func(res source.TranscribeResult) {
		if res.Length > 0 {
			fmt.Fprintf(out, "  %s of %s\r",
				transcript.Stamp(res.Heard), transcript.Stamp(res.Length))
			shown = true
		}
	}
	res, err := transcribe.Execute(ctx, v, args[1])
	if err != nil {
		return err
	}
	if shown {
		fmt.Fprintln(out)
	}

	switch {
	case res.Busy:
		fmt.Fprintf(out, "%s is already being listened to, and nothing was done\n", res.Path)
	case res.Unopened:
		fmt.Fprintf(out, "%s is not a recording anything here can open, and that is what was written\n", res.Path)
	case res.Silent:
		fmt.Fprintf(out, "%s carries no speech, and that is what was written\n", res.Path)
	default:
		fmt.Fprintf(out, "transcribed %s of %s in %s\n",
			transcript.Stamp(res.Heard), res.Path, time.Since(started).Round(time.Second))
	}
	return nil
}
