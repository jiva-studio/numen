package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// NoteQueries answers questions about notes in shapes that are not notes.
type NoteQueries interface {
	// Fingerprints is what the index believes about each file, keyed by path, so
	// a scan can decide what to reparse without reading anything.
	Fingerprints(ctx context.Context, vaultID string) (map[string]domain.FileRef, error)
	Search(ctx context.Context, vaultID, query string, limit int) ([]domain.NoteMatch, error)
	Summary(ctx context.Context, vaultID string) (domain.VaultSummary, error)
}

// LinkQueries answers what points where. It is separate from NoteQueries
// because a use case about links has no business being handed a search.
type LinkQueries interface {
	// Links returns what one note points at, resolved as of now.
	Links(ctx context.Context, vaultID, from string) ([]domain.ResolvedLink, error)
	// Backlinks returns what points at one note, by whichever address form was
	// written: its identifier, or a name that resolves to it.
	Backlinks(ctx context.Context, vaultID, to string) ([]domain.ResolvedLink, error)
}

// ProblemQueries reports what a vault contains that could not be acted on.
type ProblemQueries interface {
	Problems(ctx context.Context, vaultID string) ([]domain.VaultProblem, error)
}
