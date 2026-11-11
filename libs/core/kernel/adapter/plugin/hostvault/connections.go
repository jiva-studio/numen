package hostvault

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

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
	return c.Reader.GetConnection(ctx, v, first, second, kind)
}

// SaveConnection saves the connection.
func (c Connections) SaveConnection(ctx context.Context, connection domain.Connection) error {
	v, err := c.openConnections(c.Writer == nil)
	if err != nil {
		return err
	}
	return c.session.notifyAfterWrite(c.Writer.SaveConnection(ctx, v, connection))
}

// RemoveSide removes one side of a connection.
func (c Connections) RemoveSide(ctx context.Context, from, to, kind string) error {
	v, err := c.openConnections(c.Writer == nil)
	if err != nil {
		return err
	}
	return c.session.notifyAfterWrite(c.Writer.RemoveSide(ctx, v, from, to, kind))
}

// RemoveConnection removes the connection of the two notes.
func (c Connections) RemoveConnection(ctx context.Context, first, second, kind string) error {
	v, err := c.openConnections(c.Writer == nil)
	if err != nil {
		return err
	}
	return c.session.notifyAfterWrite(c.Writer.RemoveConnection(ctx, v, first, second, kind))
}

// Join joins the two notes.
func (c Connections) Join(ctx context.Context, from, to string, side domain.Side) error {
	v, err := c.openConnections(c.Joiner == nil)
	if err != nil {
		return err
	}
	return c.session.notifyAfterWrite(c.Joiner.Join(ctx, v, from, to, side))
}

// CreateInRelation creates a note in a relationship to another and returns its path.
func (c Connections) CreateInRelation(ctx context.Context, in domain.RelatedNote) (string, error) {
	v, err := c.openConnections(c.Creator == nil)
	if err != nil {
		return "", err
	}
	path, err := c.Creator.CreateInRelation(ctx, v, in)
	if path != "" {
		c.session.notifyFileWritten()
	}
	return path, err
}
