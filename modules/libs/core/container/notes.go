package container

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// Notes is everything that acts on the notes of a vault. A window and the tools
// an agent calls are served the same set, so what one of them refuses the other
// refuses, and a dependency named once is named for both.
type Notes struct {
	Read    note.Read
	Write   note.Write
	Create  note.Create
	Replace note.Replace
	Linking note.EditLinks
	Move    note.Move
	Rename  note.Rename
	Remove  note.Remove

	Links         note.ShowLinks
	Neighbourhood note.ShowNeighbourhood
}

// Notes builds them against this installation's vault readers and writers.
//
// index brings what a write touched up to date before it answers, so a caller
// that writes a note and searches for it in the next breath finds it.
func (c Config) Notes(
	queries port.NoteQueries,
	links port.LinkQueries,
	sources port.SourceRepository,
	known port.SourceQueries,
	index note.Levels,
) Notes {
	readers := c.VaultReaders()
	writers := c.VaultWriters()
	now := c.Clock()

	// One note.Move settles every note that travelled, whether a rename sent it
	// or a move did.
	moving := note.NewMove(readers, writers, links, queries, sources, index, now)
	moving.Sync = c.SyncSetting()

	return Notes{
		Read:    note.Read{Readers: readers},
		Write:   note.NewWrite(readers, writers, index, now),
		Create:  note.NewCreate(writers, queries, index, now),
		Replace: note.NewReplace(readers, writers, index, now),
		Linking: note.NewEditLinks(readers, writers, index, now),
		Move:    moving,
		Rename:  note.NewRename(moving),
		Remove:  note.NewRemove(writers, links, known, index),

		Links:         note.ShowLinks{Links: links},
		Neighbourhood: note.ShowNeighbourhood{Links: links, Notes: queries},
	}
}

// Following tells a window where each note went, so whoever is showing one at
// the name it had follows it to the name it now has. Every build with a window
// binds it.
func (n Notes) Following(view port.Window) Notes {
	moving := n.Move
	moving.Moving = func(ctx context.Context, went domain.Move) {
		_ = view.Moved(ctx, went)
	}
	n.Move, n.Rename = moving, note.NewRename(moving)
	return n
}

// Drawing tells a window what each write is doing while it is being made, so a
// person watching the note sees the stretch that is changing.
//
// Only a caller that is not the person binds it. The window's own writes draw
// nothing, because the person is looking at the text they typed.
func (n Notes) Drawing(view port.Window) Notes {
	tell := note.TellEditing(func(ctx context.Context, said domain.Edit) {
		_ = view.Editing(ctx, said)
	})
	n.Write.Telling, n.Replace.Telling = tell, tell
	return n
}
