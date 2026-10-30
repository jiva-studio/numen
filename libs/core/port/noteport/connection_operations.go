package noteport

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

type connectionReader interface {
	GetConnection(ctx context.Context, v domain.Vault, a, b, kind string) (domain.Connection, error)
	GetBacklinks(ctx context.Context, v domain.Vault, path string) ([]domain.Backlink, error)
}

type connectionWriter interface {
	SaveConnection(ctx context.Context, v domain.Vault, c domain.Connection) error
	RemoveSide(ctx context.Context, v domain.Vault, from, to, kind string) error
	RemoveConnection(ctx context.Context, v domain.Vault, a, b, kind string) error
	Join(ctx context.Context, v domain.Vault, from, to string, side domain.Side) error
}

// ConnectionOperations reads and writes the relationships between notes. A
// surface that serves plugins is handed it whole, so the rules of a connection
// are enforced in one place for every caller.
type ConnectionOperations interface {
	connectionReader
	connectionWriter
	// CreateInRelation creates a note joined to an existing one and returns the
	// path it is filed under. The path is set when the note exists even if the
	// error is not nil.
	CreateInRelation(ctx context.Context, v domain.Vault, in domain.RelatedNote) (string, error)
}
