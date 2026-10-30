package cli

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/jiva-studio/numen/libs/core/usecase/search"
)

// excerptRunes is how much of a passage one line of terminal output carries.
const excerptRunes = 160

func runSearchCommand(
	ctx context.Context, out, errOut io.Writer, deps Deps, args []string,
) (err error) {
	if len(args) < 2 {
		return errors.New("usage: numen-cli search <vault> <query>")
	}
	v, err := findVault(deps, args[0])
	if err != nil {
		return err
	}

	// The same search the window runs. An installation with no model answers by
	// words alone, and says nothing about it: half a search is a whole answer.
	diagnostics := newReporter(errOut)
	errorHandler := func(err error) { diagnostics.Printf("answering by words alone: %v\n", err) }
	open, err := deps.Search(ctx, v, errorHandler)
	if err != nil {
		return err
	}
	defer closeInto(&err, open.Close)
	if open.Words != nil {
		diagnostics.Printf("searching by words alone: %v\n", open.Words)
	}

	found, err := open.Search.Execute(ctx, v, args[1], search.Parameters{})
	if err != nil {
		return err
	}
	r := newReporter(out)
	if len(found) == 0 {
		r.Println("nothing found")
		return errors.Join(r.Err(), diagnostics.Err())
	}
	for _, p := range found {
		where := p.Source
		if p.Location != "" {
			where += " · " + p.Location
		}
		r.Printf("%s\n  %s\n", where, getExcerpt(p.Text))
	}
	return errors.Join(r.Err(), diagnostics.Err())
}

// getExcerpt is a passage on one line: the words, with the shape of the file taken
// out of them.
func getExcerpt(text string) string {
	words := strings.Join(strings.Fields(text), " ")
	runes := []rune(words)
	if len(runes) <= excerptRunes {
		return words
	}
	return string(runes[:excerptRunes]) + "…"
}
