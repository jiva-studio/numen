package noteport

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// ConnectionReader reads one relationship between two notes.
type ConnectionReader interface {
	GetConnection(ctx context.Context, v domain.Vault, a, b, kind string) (domain.Connection, error)
}

// ConnectionWriter saves and removes relationships between notes.
type ConnectionWriter interface {
	SaveConnection(ctx context.Context, v domain.Vault, c domain.Connection) error
	RemoveSide(ctx context.Context, v domain.Vault, from, to, kind string) error
	RemoveConnection(ctx context.Context, v domain.Vault, a, b, kind string) error
}

// ConnectionJoiner joins two notes on one side of a relationship.
type ConnectionJoiner interface {
	Join(ctx context.Context, v domain.Vault, from, to string, side domain.Side) error
}

// RelatedNoteCreator creates a note joined to an existing one.
type RelatedNoteCreator interface {
	// CreateInRelation returns the path the note is filed under. The path is set
	// when the note exists even if the error is not nil.
	CreateInRelation(ctx context.Context, v domain.Vault, in domain.RelatedNote) (string, error)
}
