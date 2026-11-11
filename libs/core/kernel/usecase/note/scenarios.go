package note

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// ConnectionReader reads one relationship between two notes.
type ConnectionReader struct {
	get connectionGetting
}

// connectionGetting reads the records two notes write about each other.
type connectionGetting interface {
	Execute(ctx context.Context, v domain.Vault, a, b, kind string) (domain.Connection, error)
}

// ConnectionWriter saves and removes relationships between notes.
type ConnectionWriter struct {
	save   connectionSaving
	side   sideRemoving
	remove connectionRemoving
}

// ConnectionJoiner joins two notes on one side of a relationship.
type ConnectionJoiner struct {
	join Join
}

// RelatedNoteCreator creates a note joined to an existing one.
type RelatedNoteCreator struct {
	create CreateInRelation
}

// NewConnectionReader takes the use case that reads a connection.
func NewConnectionReader(get connectionGetting) ConnectionReader {
	return ConnectionReader{get: get}
}

// NewConnectionWriter takes the use cases that write connections.
func NewConnectionWriter(save SaveConnection, side RemoveSide, remove RemoveConnection) ConnectionWriter {
	return ConnectionWriter{save: save, side: side, remove: remove}
}

// NewConnectionJoiner takes the use case that joins two notes.
func NewConnectionJoiner(join Join) ConnectionJoiner { return ConnectionJoiner{join: join} }

// NewRelatedNoteCreator takes the use case that creates a related note.
func NewRelatedNoteCreator(create CreateInRelation) RelatedNoteCreator {
	return RelatedNoteCreator{create: create}
}

func (c ConnectionReader) GetConnection(ctx context.Context, v domain.Vault, a, b, kind string) (domain.Connection, error) {
	return c.get.Execute(ctx, v, a, b, kind)
}

func (c ConnectionWriter) SaveConnection(ctx context.Context, v domain.Vault, conn domain.Connection) error {
	return c.save.Execute(ctx, v, conn)
}

func (c ConnectionWriter) RemoveSide(ctx context.Context, v domain.Vault, from, to, kind string) error {
	return c.side.Execute(ctx, v, from, to, kind)
}

func (c ConnectionWriter) RemoveConnection(ctx context.Context, v domain.Vault, a, b, kind string) error {
	return c.remove.Execute(ctx, v, a, b, kind)
}

func (c ConnectionJoiner) Join(ctx context.Context, v domain.Vault, from, to string, side domain.Side) error {
	return c.join.Execute(ctx, v, from, to, side)
}

func (c RelatedNoteCreator) CreateInRelation(ctx context.Context, v domain.Vault, in domain.RelatedNote) (string, error) {
	made, err := c.create.Execute(ctx, v, in)
	return made.Path, err
}

// connectionSaving makes the records between two notes exactly the ones given.
type connectionSaving interface {
	Execute(ctx context.Context, v domain.Vault, c domain.Connection) error
}

// sideRemoving removes the record one note writes about another.
type sideRemoving interface {
	Execute(ctx context.Context, v domain.Vault, from, to, kind string) error
}

// connectionRemoving removes the records of one type that two notes write about each other.
type connectionRemoving interface {
	Execute(ctx context.Context, v domain.Vault, a, b, kind string) error
}
