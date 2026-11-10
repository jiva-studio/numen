package note

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/windowport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
)

// Scenarios is everything that acts on the notes of a vault. The window and the
// agent tools use the same set, so both enforce the same rules and a dependency
// is configured once for both.
type Scenarios struct {
	Read       Read
	Write      Write
	Create     Create
	Replace    Replace
	Linking    EditLinks
	AddLinks   AddLinks
	Move       Move
	Rename     Rename
	Remove     Remove
	EmptyTrash EmptyTrash

	SaveConnection   SaveConnection
	RemoveSide       RemoveSide
	RemoveConnection RemoveConnection
	Join             Join
	CreateInRelation CreateInRelation

	Links        query.ShowLinks
	Connection   query.GetConnection
	Backlinks    query.GetBacklinks
	Neighborhood query.ShowNeighbourhood
	ResolveLinks *query.ResolveLinks

	GetTitles      query.GetTitles
	GetTitle       *query.GetTitle
	IsVaultIndexed *query.IsVaultIndexed
	GetLinkAddress *query.GetLinkAddress
	RemoveNote     RemoveNote
}

// FollowMoves notifies a window of each note move, so a view showing a note under
// its old name switches to the new name. Every build with a window
// binds it.
func (n Scenarios) FollowMoves(view windowport.Window) Scenarios {
	mover := n.Move
	mover.Drawing = view.ShowMove
	n.Move, n.Rename = mover, NewRename(mover)
	return n
}

// ShowEdits notifies a window of each write in progress, so a user watching the
// note sees the changing range.
//
// Only a caller other than the user binds it. The window's own writes are not
// drawn, because the user is looking at the text they typed.
func (n Scenarios) ShowEdits(view windowport.Window) Scenarios {
	tell := TellEdit(view.ShowEdit)
	n.Write.Drawing, n.Replace.Drawing = tell, tell
	return n
}

// ConnectionReader reads one relationship between two notes.
type ConnectionReader struct {
	get query.GetConnection
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

// NewConnectionReader takes the use case of a set of scenarios that reads a connection.
func NewConnectionReader(n Scenarios) ConnectionReader { return ConnectionReader{get: n.Connection} }

// NewConnectionWriter takes the use cases of a set of scenarios that write connections.
func NewConnectionWriter(n Scenarios) ConnectionWriter {
	return ConnectionWriter{save: n.SaveConnection, side: n.RemoveSide, remove: n.RemoveConnection}
}

// NewConnectionJoiner takes the use case of a set of scenarios that joins two notes.
func NewConnectionJoiner(n Scenarios) ConnectionJoiner { return ConnectionJoiner{join: n.Join} }

// NewRelatedNoteCreator takes the use case of a set of scenarios that creates a related note.
func NewRelatedNoteCreator(n Scenarios) RelatedNoteCreator {
	return RelatedNoteCreator{create: n.CreateInRelation}
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
