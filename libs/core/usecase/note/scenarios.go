package note

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/windowport"
)

// Scenarios is everything that acts on the notes of a vault. A window and the
// tools an agent calls are served the same set, so what one of them refuses the
// other refuses, and a dependency named once is named for both.
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

	Links         ShowLinks
	Neighbourhood ShowNeighbourhood

	GetTitles      GetTitles
	GetTitle       *GetTitle
	IsVaultIndexed *IsVaultIndexed
	GetLinkAddress *GetLinkAddress
	RemoveNote     RemoveNote
}

// FollowMoves tells a window where each note went, so whoever is showing one at
// the name it had follows it to the name it now has. Every build with a window
// binds it.
func (n Scenarios) FollowMoves(view windowport.Window) Scenarios {
	mover := n.Move
	mover.Drawing = func(ctx context.Context, went domain.Move) {
		_ = view.ShowMove(ctx, went)
	}
	n.Move, n.Rename = mover, NewRename(mover)
	return n
}

// ShowEdits tells a window what each write is doing while it is being made, so a
// person watching the note sees the stretch that is changing.
//
// Only a caller that is not the person binds it. The window's own writes draw
// nothing, because the person is looking at the text they typed.
func (n Scenarios) ShowEdits(view windowport.Window) Scenarios {
	tell := TellEdit(func(ctx context.Context, response domain.Edit) {
		_ = view.ShowEdit(ctx, response)
	})
	n.Write.Drawing, n.Replace.Drawing = tell, tell
	return n
}
