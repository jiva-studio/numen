package check

import (
	"context"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/problemport"
)

// ambiguousCheck finds links that match more than one note.
//
// Such a link is not broken: it resolves to the nearest match, which is a defined
// result. It is worth reporting because "the nearest" depends on where the
// notes currently are, so the link changes meaning when either note is
// moved, with no change to the link itself.
//
// The problem belongs to the note that contains the link. That is the file
// the user opens to resolve it, by naming which note was meant. Neither of the
// candidate notes is responsible.
type ambiguousCheck struct {
	queries problemport.ProblemQueries
}

func (ambiguousCheck) Name() domain.Check { return domain.CheckAmbiguous }
func (ambiguousCheck) IsOptIn() bool      { return false }

func (c ambiguousCheck) FindProblems(ctx context.Context, v domain.Vault) ([]domain.VaultProblem, error) {
	found, err := c.queries.GetAmbiguousLinks(ctx, v.ID)
	if err != nil {
		return nil, err
	}

	out := make([]domain.VaultProblem, 0, len(found))
	for _, l := range found {
		out = append(out, domain.VaultProblem{
			Path:       l.From,
			Kind:       domain.CheckAmbiguous,
			Target:     l.Target,
			Candidates: l.Candidates,
			Detail: fmt.Sprintf(
				"%s answers for %d notes, so this link reaches %s and would reach another if either moved: %s",
				l.Target.Value, len(l.Candidates), l.To, strings.Join(l.Candidates, ", "),
			),
		})
	}
	return out, nil
}
