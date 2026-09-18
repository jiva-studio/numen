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

// recogniseCommand reads one scanned document with a model.
//
// Nothing starts this on its own. Whether a document's own text layer is any
// good cannot be told from the text, so the layer is used until a person says
// otherwise, and this is how they say it.
func recogniseCommand(ctx context.Context, out io.Writer, deps Deps, args []string) error {
	if len(args) != 2 {
		return errors.New("usage: numen-cli recognise <vault> <file>")
	}
	v, err := findVault(deps, args[0])
	if err != nil {
		return err
	}

	// A terminal is where waiting is what a person came for, so the opener
	// fetches what is missing and waits for it. It is said out loud first: a
	// hundred and sixty megabytes is minutes, and a program that prints nothing
	// for minutes looks broken.
	fetching := func() {
		fmt.Fprintln(out, "fetching what is needed to read scans, about 160 MB")
	}
	open, err := deps.Recognise(ctx, v, fetching)
	if err != nil {
		return fmt.Errorf("nothing to read with: %w", err)
	}
	defer closeIfOpen(open.Close)

	recognise, cut := open.Recognise, open.Cut
	fmt.Fprintf(out, "reading %s with %s\n", args[1], recognise.By.Recognition())
	started := time.Now()

	// The line of pages is closed once it stops, so what follows it stands on a
	// line of its own.
	shown := false
	recognise.Cut = func(ctx context.Context, v domain.Vault, path string) error {
		_, err := cut.ExtractOne(ctx, v, path)
		return err
	}
	// A terminal that prints nothing for an hour looks broken, and this takes
	// about that. The line rewrites itself.
	recognise.OnProgress = func(res source.RecogniseResult) {
		if res.Pages > 0 {
			fmt.Fprintf(out, "  page %d of %d\r", res.Read, res.Pages)
			shown = true
		}
	}
	res, err := recognise.Execute(ctx, v, args[1])
	if err != nil {
		return err
	}
	if shown {
		fmt.Fprintln(out)
	}

	switch {
	case res.IsBusy:
		fmt.Fprintf(out, "%s is already being read, and nothing was done\n", res.Path)
	case res.IsEmpty:
		fmt.Fprintf(out, "%s says nothing that could be read, and nothing was written\n", res.Path)
	default:
		fmt.Fprintf(out, "read %d pages of %s in %s\n",
			res.Read, res.Path, time.Since(started).Round(time.Second))
	}
	return nil
}
