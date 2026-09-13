package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// Notes is what everything that acts on the notes of a vault is called where
// it is built. An application is handed the set and names it here, so naming it
// is not reaching for a scenario.
type Notes = note.Scenarios

// Notes builds everything that acts on the notes of a vault, against this
// installation's vault readers and writers.
//
// index brings what a write touched up to date before it answers, so a caller
// that writes a note and searches for it in the next breath finds it.
func (c Config) Notes(
	queries port.NoteQueries,
	links port.LinkQueries,
	sources port.SourceRepository,
	known port.SourceQueries,
	index note.Levels,
) note.Scenarios {
	readers := c.VaultReaders()
	writers := c.VaultWriters()
	now := c.Clock()

	// One note.Move settles every note that travelled, whether a rename sent it
	// or a move did.
	moving := note.NewMove(readers, writers, links, queries, sources, index, now)
	moving.Sync = c.SyncSetting()

	return note.Scenarios{
		Read:    note.NewRead(readers),
		Write:   note.NewWrite(readers, writers, index, now),
		Create:  note.NewCreate(writers, queries, index, now),
		Replace: note.NewReplace(readers, writers, index, now),
		Linking: note.NewEditLinks(readers, writers, index, now),
		Move:    moving,
		Rename:  note.NewRename(moving),
		Remove:  note.NewRemove(writers, links, known, index),

		Links:         note.NewShowLinks(links),
		Neighbourhood: note.NewShowNeighbourhood(links, queries),
	}
}
