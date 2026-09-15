package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// Cards is what everything that acts on the two notes a flashcard is made of is
// called where it is built. An application is handed the set and names it here.
type Cards = cards.Scenarios

// Cards builds everything that acts on the two notes a flashcard is made of,
// against this installation's vault readers and writers.
//
// Index brings what a write touched up to date before it answers, so a caller
// that makes a deck and lists the vault's stencils in the next breath finds
// what it made.
func (c Config) Cards(
	notes port.NoteQueries,
	links port.LinkQueries,
	index note.Levels,
) cards.Scenarios {
	readers := c.VaultReaders()
	writers := c.VaultWriters()
	now := c.Clock()

	return cards.Scenarios{
		Read:   cards.NewRead(readers, links),
		List:   cards.NewList(readers, notes),
		Write:  cards.NewWrite(readers, writers, links, index, now),
		Create: cards.NewCreate(writers, index, now),
		Rename: cards.NewRenameField(readers, writers, notes, links, index, now),
	}
}
