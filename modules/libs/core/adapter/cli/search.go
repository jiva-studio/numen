package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
)

// excerptRunes is how much of a passage one line of terminal output carries.
const excerptRunes = 160

func searchCommand(
	ctx context.Context, out, errOut io.Writer, cfg container.Config, deps Deps, args []string,
) error {
	if len(args) < 2 {
		return errors.New("usage: numen-cli search <vault> <query>")
	}
	v, err := findVault(deps, args[0])
	if err != nil {
		return err
	}
	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		return err
	}
	defer db.Close()

	// The same search the window runs. An installation with no model answers by
	// words alone, and says nothing about it: half a search is a whole answer.
	// Only the provider that embeds questions is opened: nothing here fills an
	// index.
	embedder, closeEmbedder, why := cfg.Asking(ctx)
	if why != nil {
		fmt.Fprintf(errOut, "searching by words alone: %v\n", why)
	}
	if closeEmbedder != nil {
		defer func() { _ = closeEmbedder() }()
	}

	trouble := func(err error) { fmt.Fprintf(errOut, "answering by words alone: %v\n", err) }
	found, err := cfg.Searching(db, embedder, trouble).
		Execute(ctx, v, args[1], search.Parameters{})
	if err != nil {
		return err
	}
	if len(found) == 0 {
		fmt.Fprintln(out, "nothing found")
		return nil
	}
	for _, p := range found {
		where := p.Source
		if p.Location != "" {
			where += " · " + p.Location
		}
		fmt.Fprintf(out, "%s\n  %s\n", where, excerpt(p.Text))
	}
	return nil
}

// excerpt is a passage on one line: the words, with the shape of the file taken
// out of them.
func excerpt(text string) string {
	words := strings.Join(strings.Fields(text), " ")
	runes := []rune(words)
	if len(runes) <= excerptRunes {
		return words
	}
	return string(runes[:excerptRunes]) + "…"
}
