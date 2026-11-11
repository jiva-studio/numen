package cli

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/note/domain"
)

type problemsCommand struct {
	problems func(ctx context.Context) (Problems, error)
}

func (problemsCommand) Name() string { return "problems" }

func (problemsCommand) Synopsis() string {
	return "  numen-cli problems <vault> [<check>...]      list notes with broken links, invalid frontmatter or ambiguous links"
}

func (c problemsCommand) Run(ctx context.Context, term terminal, args []string) (err error) {
	if len(args) < 1 {
		return errors.New("usage: numen-cli problems <vault> [<check>...]")
	}
	open, err := c.problems(ctx)
	if err != nil {
		return err
	}
	defer closeInto(&err, open.Close)

	var named []domain.Check
	for _, name := range args[1:] {
		named = append(named, domain.Check(name))
	}

	report, err := open.Problems.ExecuteByName(ctx, args[0], named...)
	if err != nil {
		return err
	}
	r := newReporter(term.out)
	if len(report.Found) == 0 {
		r.Println("nothing to report")
		return r.Err()
	}
	for _, p := range report.Found {
		r.Printf("%s\n  %s: %s\n", p.Path, p.Kind, p.Detail)
	}
	return r.Err()
}
