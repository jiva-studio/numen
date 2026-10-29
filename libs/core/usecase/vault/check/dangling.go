package check

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/problemport"
)

// danglingCheck finds links whose target does not exist.
//
// It is opt-in. Writing `[[Entropy]]` before the note exists is a common
// workflow, with the link serving as a reminder to create the note, and a vault
// being written contains hundreds of such links. Reported with the other
// checks they would bury the results, so this check runs only when
// requested by name.
//
// Creating the missing note resolves the problem without editing the link: the
// target of a name is resolved when the check runs.
type danglingCheck struct {
	queries problemport.ProblemQueries
}

func (danglingCheck) Name() domain.Check { return domain.CheckDangling }
func (danglingCheck) IsOptIn() bool      { return true }

func (c danglingCheck) FindProblems(ctx context.Context, v domain.Vault) ([]domain.VaultProblem, error) {
	found, err := c.queries.GetDanglingLinks(ctx, v.ID)
	if err != nil {
		return nil, err
	}

	out := make([]domain.VaultProblem, 0, len(found))
	for _, l := range found {
		out = append(out, domain.VaultProblem{
			Path:   l.From,
			Kind:   domain.CheckDangling,
			Target: l.Target,
			Detail: fmt.Sprintf("nothing in this vault answers to %s; writing the note it names is what mends it",
				l.Target.Value),
		})
	}
	return out, nil
}
