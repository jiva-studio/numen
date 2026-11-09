package vault

import (
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/windowport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
)

// Notes is what everything that acts on the notes of a vault is called where
// it is built. An application is handed the set and names it here, so naming it
// is not reaching for a scenario.
type Notes = note.Scenarios

// Notes builds everything that acts on the notes of a vault, against this
// installation's vault readers and writers.
//
// index brings what a write touched up to date before it answers, so a caller
// that writes a note and immediately searches for it finds it.
func (c Config) Notes(
	queries noteport.NoteQueries,
	links noteport.LinkQueries,
	sources sourceport.SourceRepository,
	indexed sourceport.SourceQueries,
	index note.IndexUpdater,
) note.Scenarios {
	readers := c.VaultReaders()
	writers := c.VaultWriters()
	now := c.Clock

	// One note.Move settles every note that traveled, whether a rename sent it
	// or a move did.
	mover := note.NewMove(note.MoveDeps{
		Readers: readers, Writers: writers, Links: links, Names: queries,
		Sources: sources, Index: index, Now: now,
	})
	mover.Sync = c.SyncSetting

	linking := note.NewEditLinks(readers, writers, index, now)
	remove := note.NewRemove(writers, links, indexed, index)
	addresses := note.NewGetLinkAddress(queries)
	create := note.NewCreate(writers, queries, index, now)

	return note.Scenarios{
		Read:       note.NewRead(readers),
		Write:      note.NewWrite(readers, writers, index, now),
		Create:     create,
		Replace:    note.NewReplace(readers, writers, index, now),
		Linking:    linking,
		AddLinks:   note.NewAddLinks(linking),
		Move:       mover,
		Rename:     note.NewRename(mover),
		Remove:     remove,
		EmptyTrash: note.NewEmptyTrash(writers),

		SaveConnection:   note.NewSaveConnection(readers, writers, index, now, links, addresses),
		RemoveSide:       note.NewRemoveSide(readers, writers, index, now, links, addresses),
		RemoveConnection: note.NewRemoveConnection(readers, writers, index, now, links, addresses),
		Join:             note.NewJoin(readers, writers, index, now, links, addresses),
		CreateInRelation: note.NewCreateInRelation(create, addresses),

		Links:        note.NewShowLinks(links),
		Connection:   note.NewGetConnection(links),
		Backlinks:    note.NewGetBacklinks(links),
		Neighborhood: note.NewShowNeighbourhood(links, queries),
		ResolveLinks: note.NewResolveLinks(links),

		GetTitles:      note.NewGetTitles(queries),
		GetTitle:       note.NewGetTitle(queries),
		IsVaultIndexed: note.NewIsVaultIndexed(queries),
		GetLinkAddress: addresses,
		RemoveNote:     note.NewRemoveNote(remove, readers),
	}
}

// FocusNote builds the use case that shows a note in the window.
func (c Config) FocusNote(queries noteport.NoteQueries, view windowport.Window) *note.FocusNote {
	return note.NewFocusNote(queries, view)
}

// GetNamedPaths builds the note named paths query use case.
func (Config) GetNamedPaths(queries noteport.NoteQueries) *note.GetNamedPaths {
	return note.NewGetNamedPaths(queries)
}

// Read builds the vault file reader use case.
func (c Config) Read() file.Read {
	return file.Read{Readers: c.VaultReaders()}
}
