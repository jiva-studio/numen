package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/jiva-studio/numen/modules/libs/core/transcript"
)

// importCommand fetches what is at the address a link note points at.
//
// The window asks for it as the note is made; this is the hand asking for one,
// and `--again` is how a person asks a site for its words afresh.
func importCommand(ctx context.Context, out io.Writer, deps Deps, args []string) error {
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
		return errors.New("usage: numen-cli import <vault> <note> [--again]")
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
