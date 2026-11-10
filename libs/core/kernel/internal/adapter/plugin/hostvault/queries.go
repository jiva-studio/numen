package hostvault

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

type noteRefs interface {
	GetNotes(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string]domain.NoteRef, error)
}

type noteTypes interface {
	GetTypes(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string]domain.NoteType, error)
}

type namedPaths interface {
	GetNamedPaths(ctx context.Context, vaultID domain.VaultID, name string) ([]string, error)
}

type typedPaths interface {
	GetPathsOfType(ctx context.Context, vaultID domain.VaultID, noteType domain.NoteType) ([]string, error)
}

type connectionReader interface {
	GetConnection(ctx context.Context, v domain.Vault, a, b, kind string) (domain.Connection, error)
}

type connectionWriter interface {
	SaveConnection(ctx context.Context, v domain.Vault, c domain.Connection) error
	RemoveSide(ctx context.Context, v domain.Vault, from, to, kind string) error
	RemoveConnection(ctx context.Context, v domain.Vault, a, b, kind string) error
}

type connectionJoiner interface {
	Join(ctx context.Context, v domain.Vault, from, to string, side domain.Side) error
}

type relatedNoteCreator interface {
	CreateInRelation(ctx context.Context, v domain.Vault, in domain.RelatedNote) (string, error)
}

// Connections is what the host works the relationships between notes through.
type Connections struct {
	Reader  connectionReader
	Writer  connectionWriter
	Joiner  connectionJoiner
	Creator relatedNoteCreator
}
