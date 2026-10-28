package noteport

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

type noteHeadingQueries interface {
	GetFileStats(ctx context.Context, vaultID domain.VaultID) (map[string]domain.FileStat, error)
	GetNameMatches(ctx context.Context, vaultID domain.VaultID, query string, limit int) ([]domain.NameMatch, error)
	GetHeadings(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string][]domain.Heading, error)
	GetSummary(ctx context.Context, vaultID domain.VaultID) (domain.VaultSummary, error)
}

type noteTypeQueries interface {
	GetTypes(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string]domain.NoteType, error)
	GetPathsOfType(ctx context.Context, vaultID domain.VaultID, noteType domain.NoteType) ([]string, error)
	Has(ctx context.Context, vaultID domain.VaultID) (bool, error)
}

// NoteQueries answers questions about notes in shapes that are not notes.
type NoteQueries interface {
	noteHeadingQueries
	noteTypeQueries
	GetNotes(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string]domain.NoteRef, error)
	GetOpeningNote(ctx context.Context, vaultID domain.VaultID) (domain.NoteRef, bool, error)
	GetNamedPaths(ctx context.Context, vaultID domain.VaultID, name string) ([]string, error)
}

// LinkQueries answers what points where. It is separate from NoteQueries
// because a use case about links has no business being handed a search.
type LinkQueries interface {
	GetLinks(ctx context.Context, vaultID domain.VaultID, from string) ([]domain.ResolvedLink, error)
	GetBacklinks(ctx context.Context, vaultID domain.VaultID, to string) ([]domain.ResolvedLink, error)
	ResolveLinks(ctx context.Context, vaultID domain.VaultID, from string, content []string) (map[string]domain.ResolvedLink, error)
}
