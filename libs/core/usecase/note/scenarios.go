package note

import "github.com/jiva-studio/numen/libs/core/port/windowport"

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
	ReplaceLinks ReplaceLinks
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
