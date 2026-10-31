package note

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/noteport"
	"github.com/jiva-studio/numen/libs/core/port/windowport"
)

// Scenarios is everything that acts on the notes of a vault. The window and the
// agent tools use the same set, so both enforce the same rules and a dependency
// is configured once for both.
type Scenarios struct {
	Read         Read
	Write        Write
	Create       Create
	Replace      Replace
	Linking      EditLinks
	AddLinks     AddLinks
	Move         Move
	Rename       Rename
	Remove       Remove

	SaveConnection   SaveConnection
	RemoveSide       RemoveSide
	RemoveConnection RemoveConnection
	Join             Join
	CreateInRelation CreateInRelation

	Links         ShowLinks
	Connection    GetConnection
	Backlinks     GetBacklinks
	Neighbourhood ShowNeighbourhood
	ResolveLinks  *ResolveLinks

	GetTitles      GetTitles
	GetTitle       *GetTitle
	IsVaultIndexed *IsVaultIndexed
	GetLinkAddress *GetLinkAddress
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
