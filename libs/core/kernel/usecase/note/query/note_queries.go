package query

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// Headings reads the headings of notes.
type Headings interface {
	GetHeadings(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string][]domain.Heading, error)
}

// NamedPaths finds the paths of the notes that share one name.
type NamedPaths interface {
	ListNamedPaths(ctx context.Context, vaultID domain.VaultID, name string) ([]string, error)
}

// OpeningNote finds the note a vault opens on, if it has one.
type OpeningNote interface {
	GetOpeningNote(ctx context.Context, vaultID domain.VaultID) (domain.NoteRef, bool, error)
}

// VaultHolds checks the index for any note of a vault.
type VaultHolds interface {
	Has(ctx context.Context, vaultID domain.VaultID) (bool, error)
}
