package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"

	"github.com/jiva-studio/numen/modules/libs/core/port/noteport"
	"github.com/jiva-studio/numen/modules/libs/core/port/sourceport"
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
	queries noteport.NoteQueries,
	links noteport.LinkQueries,
	sources sourceport.SourceRepository,
	indexed sourceport.SourceQueries,
	index note.Levels,
) note.Scenarios {
	readers := c.VaultReaders()
	writers := c.VaultWriters()
	now := c.Clock()

	// One note.Move settles every note that travelled, whether a rename sent it
	// or a move did.
	mover := note.NewMove(readers, writers, links, queries, sources, index, now)
	mover.Sync = c.SyncSetting()

	linking := note.NewEditLinks(readers, writers, index, now)
	remove := note.NewRemove(writers, links, indexed, index)

	return note.Scenarios{
		Read:         note.NewRead(readers),
		Write:        note.NewWrite(readers, writers, index, now),
		Create:       note.NewCreate(writers, queries, index, now),
		Replace:      note.NewReplace(readers, writers, index, now),
		Linking:      linking,
		AddLinks:     note.NewAddLinks(linking),
		ReplaceLinks: note.NewReplaceLinks(linking, note.NewGetLinkAddress(queries)),
		Move:         mover,
		Rename:       note.NewRename(mover),
		Remove:       remove,

		Links:         note.NewShowLinks(links),
		Neighbourhood: note.NewShowNeighbourhood(links, queries),

		GetTitles:      note.NewGetTitles(queries),
		GetTitle:       note.NewGetTitle(queries),
		IsVaultIndexed: note.NewIsVaultIndexed(queries),
		GetLinkAddress: note.NewGetLinkAddress(queries),
		RemoveNote:     note.NewRemoveNote(remove, readers),
	}
}
