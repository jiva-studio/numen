package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/transcript"
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
	defer closing(open.Close)

	fetch := open.ImportURL
	fetch.Again = again
	if copying {
		return copied(ctx, out, fetch, v, args[1])
	}
	fmt.Fprintf(out, "fetching what %s points at\n", args[1])

	res, err := fetch.Execute(ctx, v, args[1])
	if err != nil {
		return err
	}
	switch {
	case res.Busy:
		fmt.Fprintf(out, "%s is already being fetched, and nothing was done\n", res.Path)
	case res.Nothing:
		fmt.Fprintf(out, "%s publishes none of what was asked for, and that is what was written\n",
			res.Title)
	default:
		fmt.Fprintf(out, "%s: %d bytes of %s", res.Title, res.Words, res.Producer)
		if res.Length > 0 {
			fmt.Fprintf(out, ", %s long", transcript.Stamp(res.Length))
		}
		fmt.Fprintln(out)
	}
	return nil
}

// copied fetches a copy of the video an address names, onto this disk.
func copied(
	ctx context.Context,
	out io.Writer,
	fetch source.ImportURL,
	v domain.Vault,
	path string,
) error {
	fmt.Fprintf(out, "fetching a copy of what %s points at\n", path)
	res, err := fetch.Copy(ctx, v, path)
	if err != nil {
		return err
	}
	switch {
	case res.Busy:
		fmt.Fprintf(out, "%s is already being fetched, and nothing was done\n", res.Path)
	case res.TooLarge():
		fmt.Fprintf(out, "%s would take %s, over the %s a copy may be\n",
			res.Path, sized(res.Bytes), sized(res.Limit))
	case res.Existed:
		fmt.Fprintf(out, "a copy of %s is already here\n", sized(res.Bytes))
	default:
		fmt.Fprintf(out, "a copy of %s is here\n", sized(res.Bytes))
	}
	return nil
}

// sized is how large something is, in the unit a person reads it in.
func sized(bytes int64) string {
	if bytes < 1<<20 {
		return fmt.Sprintf("%d bytes", bytes)
	}
	return fmt.Sprintf("%d MB", bytes>>20)
}
