package noteport

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

type NoteTypes interface {
	GetTypes(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string]domain.NoteType, error)
}

// NoteRefs reads what is needed to display notes. A path that matches no note
// is absent from the result.
type NoteRefs interface {
	GetNotes(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string]domain.NoteRef, error)
}

// LinkQueries answers what points where.
type LinkQueries interface {
	ListLinks(ctx context.Context, vaultID domain.VaultID, from string) ([]domain.ResolvedLink, error)
	ListBacklinks(ctx context.Context, vaultID domain.VaultID, to string) ([]domain.ResolvedLink, error)
	ResolveLinks(ctx context.Context, vaultID domain.VaultID, from string, content []string) (map[string]domain.ResolvedLink, error)
}

// LinkBatches answers LinkQueries for many notes in one call of the caller, so a caller that
// walks a graph asks once per step; an implementation may still read path by path. A path with no links is absent from the result.
type LinkBatches interface {
	ListLinksOf(ctx context.Context, vaultID domain.VaultID, from []string) (map[string][]domain.ResolvedLink, error)
	ListBacklinksOf(ctx context.Context, vaultID domain.VaultID, to []string) (map[string][]domain.ResolvedLink, error)
}
