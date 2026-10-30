package note

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/noteport"
)

// Connections is the set of use cases that read and write the relationships
// between notes, as one value a surface can be handed.
type Connections struct {
	get      GetConnection
	save     SaveConnection
	side     RemoveSide
	remove   RemoveConnection
	join     Join
	create   CreateInRelation
	backlink GetBacklinks
}

var _ noteport.ConnectionOperations = Connections{}

// NewConnections groups the connection use cases of a set of scenarios.
func NewConnections(n Scenarios) Connections {
	return Connections{
		get: n.Connection, save: n.SaveConnection, side: n.RemoveSide, remove: n.RemoveConnection,
		join: n.Join, create: n.CreateInRelation, backlink: n.Backlinks,
	}
}

func (c Connections) GetConnection(ctx context.Context, v domain.Vault, a, b, kind string) (domain.Connection, error) {
	return c.get.Execute(ctx, v, a, b, kind)
}

func (c Connections) SaveConnection(ctx context.Context, v domain.Vault, conn domain.Connection) error {
	return c.save.Execute(ctx, v, conn)
}

func (c Connections) RemoveSide(ctx context.Context, v domain.Vault, from, to, kind string) error {
	return c.side.Execute(ctx, v, from, to, kind)
}

func (c Connections) RemoveConnection(ctx context.Context, v domain.Vault, a, b, kind string) error {
	return c.remove.Execute(ctx, v, a, b, kind)
}

func (c Connections) Join(ctx context.Context, v domain.Vault, from, to string, side domain.Side) error {
	return c.join.Execute(ctx, v, from, to, side)
}

func (c Connections) CreateInRelation(ctx context.Context, v domain.Vault, in domain.RelatedNote) (string, error) {
	made, err := c.create.Execute(ctx, v, in)
	return made.Path, err
}

func (c Connections) GetBacklinks(ctx context.Context, v domain.Vault, path string) ([]domain.Backlink, error) {
	return c.backlink.Execute(ctx, v, path)
}
