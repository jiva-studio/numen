package cli

import (
	"context"
	"errors"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
)

// excerptRunes is how much of a search result one line of terminal output carries.
const excerptRunes = 160

type searchCommand struct {
	session Session
}

func (searchCommand) Name() string { return "search" }

func (searchCommand) Synopsis() string {
	return "  numen-cli search <vault> <query>             full-text search within one vault"
}

func (c searchCommand) Run(ctx context.Context, term terminal, args []string) error {
	if len(args) < 2 {
		return errors.New("usage: numen-cli search <vault> <query>")
	}

	// The same search the window runs. An installation with no model searches by
	// words alone, which is reported through the error handler.
	diagnostics := newReporter(term.errOut)
	errorHandler := func(err error) { diagnostics.Printf("search by words only: %v\n", err) }
	indexed, err := c.session.OpenIndexed(ctx, IndexOptions{ErrorHandler: errorHandler})
	if err != nil {
		return err
	}
	open := indexed.Search
	if open.Words != nil {
		diagnostics.Printf("search by words only: %v\n", open.Words)
	}

	found, err := open.Search.ExecuteByName(ctx, args[0], args[1], search.Parameters{})
	if err != nil {
		return err
	}
	r := newReporter(term.out)
	if len(found) == 0 {
		r.Println("nothing found")
		return errors.Join(r.Err(), diagnostics.Err())
	}
	for _, p := range found {
		where := p.Path
		if p.PlaceLabel != "" {
			where += " · " + p.PlaceLabel
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
