package hostvault

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
)

type connectionGetter interface {
	Execute(ctx context.Context, v domain.Vault, a, b, kind string) (domain.Connection, error)
}

type connectionSaver interface {
	Execute(ctx context.Context, v domain.Vault, c domain.Connection) error
}

type sideRemover interface {
	Execute(ctx context.Context, v domain.Vault, from, to, kind string) error
}

type connectionRemover interface {
	Execute(ctx context.Context, v domain.Vault, a, b, kind string) error
}

type noteJoiner interface {
	Execute(ctx context.Context, v domain.Vault, from, to string, side domain.Side) error
}

type relatedNoteCreator interface {
	Execute(ctx context.Context, v domain.Vault, in domain.RelatedNote) (note.CreateResult, error)
}

// Connections is what the host works the relationships between notes through. A call whose use case is
// nil fails with plugin.ErrNoConnections.
type Connections struct {
	Reader      connectionGetter
	Saver       connectionSaver
	SideRemover sideRemover
	Remover     connectionRemover
	Joiner      noteJoiner
	Creator     relatedNoteCreator

	session sessionAccess
}

func (c Connections) openConnections(isMissing bool) (domain.Vault, error) {
	v, err := c.session.open()
	if err != nil {
		return domain.Vault{}, err
	}
	if isMissing {
		return domain.Vault{}, plugin.ErrNoConnections
	}
	return v, nil
}

// GetConnection returns the connection of the two notes.
func (c Connections) GetConnection(ctx context.Context, first, second, kind string) (domain.Connection, error) {
	v, err := c.openConnections(c.Reader == nil)
	if err != nil {
		return domain.Connection{}, err
	}
	return c.Reader.Execute(ctx, v, first, second, kind)
}

// SaveConnection saves the connection.
func (c Connections) SaveConnection(ctx context.Context, connection domain.Connection) error {
	v, err := c.openConnections(c.Saver == nil)
	if err != nil {
		return err
	}
	return c.session.notifyAfterWrite(c.Saver.Execute(ctx, v, connection))
}

// RemoveSide removes one side of a connection.
func (c Connections) RemoveSide(ctx context.Context, from, to, kind string) error {
	v, err := c.openConnections(c.SideRemover == nil)
	if err != nil {
		return err
	}
	return c.session.notifyAfterWrite(c.SideRemover.Execute(ctx, v, from, to, kind))
}

// RemoveConnection removes the connection of the two notes.
func (c Connections) RemoveConnection(ctx context.Context, first, second, kind string) error {
	v, err := c.openConnections(c.Remover == nil)
	if err != nil {
		return err
	}
	return c.session.notifyAfterWrite(c.Remover.Execute(ctx, v, first, second, kind))
}

// Join joins the two notes.
func (c Connections) Join(ctx context.Context, from, to string, side domain.Side) error {
	v, err := c.openConnections(c.Joiner == nil)
	if err != nil {
		return err
	}
	return c.session.notifyAfterWrite(c.Joiner.Execute(ctx, v, from, to, side))
}

// CreateInRelation creates a note in a relationship to another and returns its path.
func (c Connections) CreateInRelation(ctx context.Context, in domain.RelatedNote) (string, error) {
	v, err := c.openConnections(c.Creator == nil)
	if err != nil {
		return "", err
	}
	made, err := c.Creator.Execute(ctx, v, in)
	if made.Path != "" {
		c.session.notifyFileWritten()
	}
	return made.Path, err
}
