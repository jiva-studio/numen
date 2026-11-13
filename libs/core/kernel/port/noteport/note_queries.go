package noteport

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// FileStats reads what the index recorded of each file of a vault, keyed by
// path, so a scan can decide what to reparse without reading anything.
type FileStats interface {
	GetFileStats(ctx context.Context, vaultID domain.VaultID) (map[string]domain.FileStat, error)
}

// NameMatches finds notes by a part of their name.
type NameMatches interface {
	GetNameMatches(ctx context.Context, vaultID domain.VaultID, query string, limit int) ([]domain.NameMatch, error)
}

// Headings reads the headings of notes.
type Headings interface {
	GetHeadings(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string][]domain.Heading, error)
}

// VaultSummaries counts what the index holds of one vault.
type VaultSummaries interface {
	GetSummary(ctx context.Context, vaultID domain.VaultID) (domain.VaultSummary, error)
}

type NoteTypes interface {
	GetTypes(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string]domain.NoteType, error)
}

// VaultHolds checks the index for any note of a vault.
type VaultHolds interface {
	Has(ctx context.Context, vaultID domain.VaultID) (bool, error)
}

// NoteRefs reads what is needed to display notes. A path that matches no note
// is absent from the result.
type NoteRefs interface {
	GetNotes(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string]domain.NoteRef, error)
}

// OpeningNote finds the note a vault opens on, if it has one.
type OpeningNote interface {
	GetOpeningNote(ctx context.Context, vaultID domain.VaultID) (domain.NoteRef, bool, error)
}

// NamedPaths finds the paths of the notes that share one name.
type NamedPaths interface {
	GetNamedPaths(ctx context.Context, vaultID domain.VaultID, name string) ([]string, error)
}

// LinkQueries answers what points where.
type LinkQueries interface {
	GetLinks(ctx context.Context, vaultID domain.VaultID, from string) ([]domain.ResolvedLink, error)
	GetBacklinks(ctx context.Context, vaultID domain.VaultID, to string) ([]domain.ResolvedLink, error)
	ResolveLinks(ctx context.Context, vaultID domain.VaultID, from string, content []string) (map[string]domain.ResolvedLink, error)
}

// LinkBatches answers LinkQueries for many notes in one call of the caller, so a caller that
// walks a graph asks once per step; an implementation may still read path by path. A path with no links is absent from the result.
type LinkBatches interface {
	GetLinksOf(ctx context.Context, vaultID domain.VaultID, from []string) (map[string][]domain.ResolvedLink, error)
	GetBacklinksOf(ctx context.Context, vaultID domain.VaultID, to []string) (map[string][]domain.ResolvedLink, error)
}
