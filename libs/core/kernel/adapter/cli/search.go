package cli

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
)

// excerptRunes is how much of a passage one line of terminal output carries.
const excerptRunes = 160

func runSearchCommand(
	ctx context.Context, deps Deps, out, errOut io.Writer, args []string,
) (err error) {
	if len(args) < 2 {
		return errors.New("usage: numen-cli search <vault> <query>")
	}

	// The same search the window runs. An installation with no model searches by
	// words alone, which is reported through the error handler.
	diagnostics := newReporter(errOut)
	errorHandler := func(err error) { diagnostics.Printf("search by words only: %v\n", err) }
	open, err := deps.Search(ctx, errorHandler)
	if err != nil {
		return err
	}
	defer closeInto(&err, open.Close)
	if open.Words != nil {
		diagnostics.Printf("search by words only: %v\n", open.Words)
	}

	found, err := open.Search.ExecuteByName(ctx, args[0], args[1], search.Parameters{})
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

func getExcerpt(text string) string {
	words := strings.Join(strings.Fields(text), " ")
	runes := []rune(words)
	if len(runes) <= excerptRunes {
		return words
	}
	return string(runes[:excerptRunes]) + "…"
}
