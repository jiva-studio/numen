package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// ProblemQueries answers the questions a check cannot answer from one file.
//
// Two of these are read back from what a scan stored while parsing; two are
// worked out across the whole vault at the moment they are asked, because what
// they answer depends on every other note and stops being true when one moves.
type ProblemQueries interface {
	// Noted is what parsing each file turned up, as the parser said it.
	Noted(ctx context.Context, vaultID string) ([]domain.VaultProblem, error)
	// Unreadable is the notes whose frontmatter is not YAML, with what the
	// parser said about it.
	Unreadable(ctx context.Context, vaultID string) ([]domain.VaultProblem, error)

	// Ambiguous is every link that more than one note answers to, resolved the
	// same way a link is resolved anywhere else.
	Ambiguous(ctx context.Context, vaultID string) ([]domain.AmbiguousLink, error)
	// Dangling is every link that reaches nothing at all.
	Dangling(ctx context.Context, vaultID string) ([]domain.ResolvedLink, error)
}
