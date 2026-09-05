package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// Cards is everything that acts on the two notes a flashcard is made of. Both
// the window and the tools an agent calls are served the same set, so what one
// of them refuses the other refuses.
type Cards struct {
	Read   cards.Read
	List   cards.List
	Write  cards.Write
	Create cards.Create
	Rename cards.RenameField
}

// Cards builds them against this installation's vault readers and writers.
//
// Index brings what a write touched up to date before it answers, so a caller
// that makes a deck and lists the vault's stencils in the next breath finds
// what it made.
func (c Config) Cards(
	notes port.NoteQueries,
	links port.LinkQueries,
	index note.Levels,
) Cards {
	readers := c.VaultReaders()
	writers := c.VaultWriters()
	now := c.Clock()

	return Cards{
		Read:   cards.NewRead(readers, links),
		List:   cards.NewList(readers, notes),
		Write:  cards.NewWrite(readers, writers, links, index, now),
		Create: cards.NewCreate(writers, index, now),
		Rename: cards.NewRenameField(readers, writers, notes, links, index, now),
	}
}
