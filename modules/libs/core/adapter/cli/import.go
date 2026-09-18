package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// importCommand fetches what is at the address a url file holds.
//
// The window asks for it as the file is made; this is the hand asking for one,
// and `--again` is how a person asks a site for its words afresh.
func importCommand(ctx context.Context, out io.Writer, deps Deps, args []string) error {
	again, copying := false, false
	rest := make([]string, 0, len(args))
	for _, one := range args {
		switch one {
		case "--again":
			again = true
		case "--copy":
			copying = true
		default:
			rest = append(rest, one)
		}
	}
	args = rest
	if len(args) != 2 {
		return errors.New("usage: numen-cli import <vault> <url> [--again] [--copy]")
	}
	v, err := findVault(deps, args[0])
	if err != nil {
		return err
	}
	open, err := deps.ImportURL(ctx, v)
	if err != nil {
		return fmt.Errorf("nothing to fetch with: %w", err)
	}
	defer closeIfOpen(open.Close)

	importing := open.ImportURL
	importing.Again = again
	if copying {
		return copyURL(ctx, out, importing, v, args[1])
	}
	fmt.Fprintf(out, "downloading what %s points at\n", args[1])

	res, err := importing.Execute(ctx, v, args[1])
	if errors.Is(err, source.ErrBeingDownloaded) {
		fmt.Fprintf(out, "%s is already being downloaded, and nothing was done\n", args[1])
		return nil
	}
	if err != nil {
		return err
	}
	if res.Nothing {
		fmt.Fprintf(out, "%s publishes none of what was asked for, and that is what was written\n",
			res.Path)
		return nil
	}
	fmt.Fprintf(out, "%s: %d bytes of %s\n", res.Path, res.Bytes, res.Producer)
	return nil
}

// copyURL fetches a copy of what an address points at, onto this disk.
func copyURL(
	ctx context.Context,
	out io.Writer,
	importing source.ImportURL,
	v domain.Vault,
	path string,
) error {
	fmt.Fprintf(out, "downloading a copy of what %s points at\n", path)
	res, err := importing.Copy(ctx, v, path)
	if errors.Is(err, source.ErrBeingDownloaded) {
		fmt.Fprintf(out, "%s is already being downloaded, and nothing was done\n", path)
		return nil
	}
	if err != nil {
		return err
	}
	switch {
	case res.IsTooLarge():
		fmt.Fprintf(out, "%s would take %s, over the %s a copy may be\n",
			res.Path, describeSize(res.Bytes), describeSize(res.Limit))
	case res.Existed:
		fmt.Fprintf(out, "a copy of %s is already here\n", describeSize(res.Bytes))
	default:
		fmt.Fprintf(out, "a copy of %s is here\n", describeSize(res.Bytes))
	}
	return nil
}

// describeSize is how large something is, in the unit a person reads it in.
func describeSize(bytes int64) string {
	if bytes < 1<<20 {
		return fmt.Sprintf("%d bytes", bytes)
	}
	return fmt.Sprintf("%d MB", bytes>>20)
}
