package note

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
)

// Scenarios is everything that acts on the notes of a vault. The window and the
// agent tools use the same set, so both enforce the same rules and a dependency
// is configured once for both.
type Scenarios struct {
	Edit    Editing
	Place   Placing
	Connect Connecting
	Refer   Referring
	Browse  Browsing
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

// Referring adds, lists and resolves the links of notes.
type Referring struct {
	AddLinks       AddLinks
	Links          query.ShowLinks
	Backlinks      query.GetBacklinks
	ResolveLinks   *query.ResolveLinks
	GetLinkAddress *query.GetLinkAddress
}

// Browsing answers questions about the notes of a vault.
type Browsing struct {
	Connection     query.GetConnection
	GetTitles      query.GetTitles
	GetTitle       *query.GetTitle
	IsVaultIndexed *query.IsVaultIndexed
}

// MoveView tells the window that a note moved.
type MoveView interface {
	ShowMove(ctx context.Context, went domain.Move) error
}

// EditView tells the window that a change to a note is in progress.
type EditView interface {
	ShowEdit(ctx context.Context, response domain.Edit) error
}

// FollowMoves notifies a window of each note move, so a view showing a note under
// its old name switches to the new name. Every build with a window
// binds it.
func (n Scenarios) FollowMoves(view MoveView) Scenarios {
	mover := n.Place.Move
	mover.Drawing = view.ShowMove
	n.Place.Move, n.Place.Rename = mover, NewRename(mover)
	return n
}

// ShowEdits notifies a window of each write in progress, so a user watching the
// note sees the changing range.
//
// Only a caller other than the user binds it. The window's own writes are not
// drawn, because the user is looking at the text they typed.
func (n Scenarios) ShowEdits(view EditView) Scenarios {
	tell := TellEdit(view.ShowEdit)
	n.Edit.Write.Drawing, n.Edit.Replace.Drawing = tell, tell
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
func NewConnectionReader(n Scenarios) ConnectionReader {
	return ConnectionReader{get: n.Browse.Connection}
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
