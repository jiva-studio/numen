package check

import (
	"context"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// ambiguousCheck is a link that reaches more than one note.
//
// It is not broken: it reaches the nearest of them, and that is a defined
// answer. What makes it worth showing is that "the nearest" is a fact about
// where the notes currently sit, so the link changes meaning when either of
// them is moved — silently, and without the link itself changing at all.
//
// The problem belongs to the note that wrote the link. That is the file
// somebody opens to settle it, by naming which note they meant. Neither of the
// notes it could mean has anything to answer for.
type ambiguousCheck struct{ queries port.ProblemQueries }

func (ambiguousCheck) Name() domain.Check { return domain.CheckAmbiguous }
func (ambiguousCheck) Quiet() bool        { return false }

func (c ambiguousCheck) Look(ctx context.Context, v domain.Vault) ([]domain.VaultProblem, error) {
	found, err := c.queries.Ambiguous(ctx, string(v.ID))
	if err != nil {
		return nil, err
	}

	out := make([]domain.VaultProblem, 0, len(found))
	for _, l := range found {
		out = append(out, domain.VaultProblem{
			Path:       l.From,
			Check:      domain.CheckAmbiguous,
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
