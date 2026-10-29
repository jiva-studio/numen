package problemport

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// ProblemQueries answers the questions that a check cannot answer from a single
// file.
//
// Two of these are read from the results stored by the scan during parsing; two
// are computed across the whole vault when queried, because their results
// depend on every other note and become invalid when any note moves.
type ProblemQueries interface {
	// GetParseProblems returns the problems found while parsing each file, as
	// reported by the parser.
	GetParseProblems(ctx context.Context, vaultID domain.VaultID) ([]domain.VaultProblem, error)
	// GetUnreadable returns the notes whose frontmatter is not YAML, with the
	// parser's error message.
	GetUnreadable(ctx context.Context, vaultID domain.VaultID) ([]domain.VaultProblem, error)

	// GetAmbiguousLinks returns every link that more than one note matches,
	// resolved the same way as links everywhere else.
	GetAmbiguousLinks(ctx context.Context, vaultID domain.VaultID) ([]domain.AmbiguousLink, error)
	// GetDanglingLinks returns every link that resolves to no note.
	GetDanglingLinks(ctx context.Context, vaultID domain.VaultID) ([]domain.ResolvedLink, error)
}
