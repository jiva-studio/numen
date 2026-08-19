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

	// Search is the notes whose text matches the words typed. A search over
	// everything the vault holds is PassageQueries and the use case above it.
	Search(ctx context.Context, vaultID, query string, limit int) ([]domain.NoteMatch, error)

	// Titles is the names in a vault that match the words typed: a note's own
	// title, and the headings inside notes. Searching the text a vault holds is
	// PassageQueries and the use case above it.
	Titles(ctx context.Context, vaultID, query string, limit int) ([]domain.TitleMatch, error)

	Summary(ctx context.Context, vaultID string) (domain.VaultSummary, error)

	// Notes returns what is needed to show a note, for the paths asked about.
	// Paths that name nothing are absent from the answer: a link resolves as
	// of now, and what it resolved to a moment ago may be gone.
	Notes(ctx context.Context, vaultID string, paths []string) (map[string]domain.NoteRef, error)

	// Opening is the note to show when nothing else has been chosen. False when
	// the vault holds none.
	Opening(ctx context.Context, vaultID string) (domain.NoteRef, bool, error)

	// Named is the paths of every note filed under one name. More than one is
	// what makes a link written by that name ambiguous.
	Named(ctx context.Context, vaultID, name string) ([]string, error)
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
