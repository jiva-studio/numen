package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/ocr/onnx"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/source"
)

// recogniseCommand reads one scanned document with a model.
//
// Nothing starts this on its own. Whether a document's own text layer is any
// good cannot be told from the text, so the layer is used until a person says
// otherwise, and this is how they say it.
func recogniseCommand(ctx context.Context, out io.Writer, cfg container.Config, args []string) error {
	if len(args) != 2 {
		return errors.New("usage: numen-cli recognise <vault> <file>")
	}
	v, err := findVault(cfg, args[0])
	if err != nil {
		return err
	}
	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		return err
	}
	defer db.Close()

	// A terminal is where waiting is what a person came for, so this fetches
	// what is missing and waits for it. It is said out loud first: a hundred and
	// sixty megabytes is minutes, and a program that prints nothing for minutes
	// looks broken.
	if !onnx.Ready(cfg.Recognition) {
		fmt.Fprintln(out, "fetching what is needed to read scans, about 160 MB")
	}
	models, closeModels, why := cfg.Recogniser()
	if why != nil {
		return fmt.Errorf("nothing to read with: %w", why)
	}
	defer closeModels()

	fmt.Fprintf(out, "reading %s with %s\n", args[1], models.Recognition())
	started := time.Now()

	recognise := source.Recognise{
		Readers: cfg.VaultReaders(),
		Sources: db.Sources(),
		Derived: cfg.DerivedStores(),
		By:      models,
		// A terminal that prints nothing for an hour looks broken, and this
		// takes about that. The line rewrites itself.
		OnProgress: func(res source.RecogniseResult) {
			if res.Pages > 0 {
				fmt.Fprintf(out, "  page %d of %d\r", res.Read, res.Pages)
			}
		},
	}
	res, err := recognise.Execute(ctx, v, args[1])
	if err != nil {
		return err
	}

	switch {
	case res.Empty:
		fmt.Fprintf(out, "%s says nothing that could be read, and nothing was written\n", res.Path)
	default:
		fmt.Fprintf(out, "read %d pages of %s in %s\n",
			res.Read, res.Path, time.Since(started).Round(time.Second))
		fmt.Fprintln(out, "it is cut into chunks by the next scan")
	}
	return nil
}
