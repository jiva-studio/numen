package check

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// dangling is a link that reaches nothing.
//
// It is quiet, and that is the whole design of it. Writing `[[Entropy]]` before
// the note exists is how people work — the link is a note to themselves that
// the note is owed — and a vault being written has hundreds of them. Reported
// alongside everything else they would bury it, so this one arrives only when
// it is asked for by name.
//
// Adding the missing note repairs it, without anyone touching the link: what a
// name reaches is worked out at the moment it is asked.
type dangling struct{ queries port.ProblemQueries }

func (dangling) Name() domain.Check { return domain.CheckDangling }
func (dangling) Quiet() bool        { return true }

func (c dangling) Look(ctx context.Context, v domain.Vault) ([]domain.VaultProblem, error) {
	found, err := c.queries.Dangling(ctx, v.ID)
	if err != nil {
		return nil, err
	}

	out := make([]domain.VaultProblem, 0, len(found))
	for _, l := range found {
		out = append(out, domain.VaultProblem{
			Path:   l.From,
			Check:  domain.CheckDangling,
			Target: l.Target,
			Detail: fmt.Sprintf("nothing in this vault answers to %s; writing the note it names is what mends it",
				l.Target.Value),
		})
	}
	return out, nil
}
