package note

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// Scenarios is everything that changes the notes of a vault, grouped; the container builds each
// use case on its own. The window and the agent tools use the same set, so both enforce the same
// rules and a dependency is configured once for both.
type Scenarios struct {
	Edit    Editing
	Place   Placing
	Connect Connecting
	Refer   Referring
}

// Editing reads and changes the text of a note.
type Editing struct {
	Read    Read
	Write   Write
	Create  Create
	Replace Replace
	Linking EditLinks
}

// Placing moves, renames and removes notes.
type Placing struct {
	Move       Move
	Rename     Rename
	Remove     Remove
	RemoveNote RemoveNote
	EmptyTrash EmptyTrash
}

// Connecting changes the relationships between notes.
type Connecting struct {
	SaveConnection   SaveConnection
	RemoveSide       RemoveSide
	RemoveConnection RemoveConnection
	Join             Join
	CreateInRelation CreateInRelation
}

// WithWrites returns Connecting with every connection write tracked through gate and follow.
func (c Connecting) WithWrites(gate WriteGate, follow IndexFollow) Connecting {
	track := NewTrackWrites(gate, follow)
	c.SaveConnection.editor.track = track
	c.RemoveSide.editor.track = track
	c.RemoveConnection.editor.track = track
	c.Join.editor.track = track
	return c
}

// Referring adds the links of notes.
type Referring struct {
	AddLinks AddLinks
}

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
	save   SaveConnection
	side   RemoveSide
	remove RemoveConnection
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

// NewConnectionWriter takes the use cases of a set of scenarios that write connections.
func NewConnectionWriter(n Scenarios) ConnectionWriter {
	return ConnectionWriter{save: n.Connect.SaveConnection, side: n.Connect.RemoveSide, remove: n.Connect.RemoveConnection}
}

// NewConnectionJoiner takes the use case of a set of scenarios that joins two notes.
func NewConnectionJoiner(n Scenarios) ConnectionJoiner { return ConnectionJoiner{join: n.Connect.Join} }

// NewRelatedNoteCreator takes the use case of a set of scenarios that creates a related note.
func NewRelatedNoteCreator(n Scenarios) RelatedNoteCreator {
	return RelatedNoteCreator{create: n.Connect.CreateInRelation}
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
