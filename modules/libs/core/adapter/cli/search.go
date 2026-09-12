package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
)

// excerptRunes is how much of a passage one line of terminal output carries.
const excerptRunes = 160

func searchCommand(
	ctx context.Context, out, errOut io.Writer, deps Deps, args []string,
) error {
	if len(args) < 2 {
		return errors.New("usage: numen-cli search <vault> <query>")
	}
	v, err := findVault(deps, args[0])
	if err != nil {
		return err
	}

	// The same search the window runs. An installation with no model answers by
	// words alone, and says nothing about it: half a search is a whole answer.
	errorHandler := func(err error) { fmt.Fprintf(errOut, "answering by words alone: %v\n", err) }
	open, err := deps.Search(ctx, errorHandler)
	if err != nil {
		return err
	}
	defer closing(open.Close)
	if open.Words != nil {
		fmt.Fprintf(errOut, "searching by words alone: %v\n", open.Words)
	}

	found, err := open.Search.Execute(ctx, v, args[1], search.Parameters{})
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
