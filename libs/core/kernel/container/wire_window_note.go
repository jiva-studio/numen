package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	notewindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
)

func mountNote(w *windowWiring) {
	state := w.inst.State
	notes := w.notes
	addresses := notewindow.NewAddresses(state, notes.GetLinkAddress)
	prose := notewindow.NewProse(state, state, addresses, &notes.Read, &notes.Write, &notes.Create, &notes.Rename)
	outline := notewindow.NewOutline(
		state, w.files.ListTabKinds,
		query.NewGetOpeningNote(w.db.OpeningNote()),
		query.NewGetTypes(w.db.NoteTypes()),
		query.NewGetHeadings(w.db.Headings()),
		&notes.Neighborhood,
	)
	links := notewindow.NewLinks(
		state, state, addresses,
		query.NewResolveLinks(w.db.Links()), &notes.Linking, &notes.CreateInRelation, &notes.Backlinks,
	)
	connections := notewindow.NewConnections(
		state, state, &notes.Connection, &notes.SaveConnection, &notes.RemoveSide, &notes.RemoveConnection, &notes.Join,
	)
	handler := notewindow.NewHandler(state, prose, outline, links, connections)
	w.inst.Mount(editor.NewNoteRoute(state, handler))
}
