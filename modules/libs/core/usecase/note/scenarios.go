package note

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Scenarios is everything that acts on the notes of a vault. A window and the
// tools an agent calls are served the same set, so what one of them refuses the
// other refuses, and a dependency named once is named for both.
type Scenarios struct {
	Read    Read
	Write   Write
	Create  Create
	Replace Replace
	Linking EditLinks
	Move    Move
	Rename  Rename
	Remove  Remove

	Links         ShowLinks
	Neighbourhood ShowNeighbourhood
}

// FollowMoves tells a window where each note went, so whoever is showing one at
// the name it had follows it to the name it now has. Every build with a window
// binds it.
func (n Scenarios) FollowMoves(view port.Window) Scenarios {
	moving := n.Move
	moving.Drawing = func(ctx context.Context, went domain.Move) {
		_ = view.ShowMove(ctx, went)
	}
	n.Move, n.Rename = moving, NewRename(moving)
	return n
}

// Drawing tells a window what each write is doing while it is being made, so a
// person watching the note sees the stretch that is changing.
//
// Only a caller that is not the person binds it. The window's own writes draw
// nothing, because the person is looking at the text they typed.
func (n Scenarios) Drawing(view port.Window) Scenarios {
	tell := TellEdit(func(ctx context.Context, said domain.Edit) {
		_ = view.ShowEdit(ctx, said)
	})
	n.Write.Drawing, n.Replace.Drawing = tell, tell
	return n
}
