package vault

import (
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/windowport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	vaultusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
)

// Notes builds everything that acts on the notes of a vault, against this
// installation's vault readers and writers.
//
// index brings what a write touched up to date before it answers, so a caller
// that writes a note and immediately searches for it finds it.
func (c Config) Notes(db *vaultusecase.Index, index note.IndexUpdater) note.Scenarios {
	names, refs, links := db.NamedPaths(), db.NoteRefs(), db.Links()
	sources, indexed := db.Sources(), db.SourcesKnown()
	readers := c.VaultReaders()
	writers := c.writers()
	now := c.Clock

	// One note.Move settles every note that traveled, whether a rename sent it
	// or a move did.
	mover := note.NewMove(note.MoveDeps{
		Readers: readers, Writers: writers, Links: links, Names: names,
		Sources: sources, Index: index, Now: now,
	})
	mover.Sync = c.SyncSetting

	linking := note.NewEditLinks(readers, writers, index, now)
	remove := note.NewRemove(writers, links, indexed, index)
	addresses := query.NewGetLinkAddress(names)
	create := note.NewCreate(writers, names, index, now)

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

		Links:        query.NewShowLinks(links),
		Connection:   query.NewGetConnection(links),
		Backlinks:    query.NewGetBacklinks(links),
		Neighborhood: query.NewShowNeighbourhood(links, refs),
		ResolveLinks: query.NewResolveLinks(links),

		GetTitles:      query.NewGetTitles(refs),
		GetTitle:       query.NewGetTitle(refs),
		IsVaultIndexed: query.NewIsVaultIndexed(db.VaultHolds()),
		GetLinkAddress: addresses,
		RemoveNote:     note.NewRemoveNote(remove, readers),
	}
}

// FocusNote builds the use case that shows a note in the window.
func (c Config) FocusNote(refs noteport.NoteRefs, view windowport.Window) *query.FocusNote {
	return query.NewFocusNote(refs, view)
}

// GetNamedPaths builds the note named paths query use case.
func (Config) GetNamedPaths(names noteport.NamedPaths) *query.GetNamedPaths {
	return query.NewGetNamedPaths(names)
}

// Read builds the vault file reader use case.
func (c Config) Read() file.Read {
	return file.Read{Readers: c.VaultReaders()}
}
