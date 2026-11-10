package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	notewindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
)

func mountNote(w *windowWiring) {
	state := w.inst.State
	notes := w.notes
	addresses := notewindow.NewAddresses(state, notes.Refer.GetLinkAddress)
	prose := notewindow.NewProse(state, state, addresses, &notes.Edit.Read, &notes.Edit.Write, &notes.Edit.Create, &notes.Place.Rename)
	outline := notewindow.NewOutline(
		state, w.files.ListTabKinds,
		query.NewGetOpeningNote(w.db.OpeningNote()),
		query.NewGetTypes(w.db.NoteTypes()),
		query.NewGetHeadings(w.db.Headings()),
		&notes.Browse.Neighborhood,
	)
	links := notewindow.NewLinks(
		state, state, addresses,
		query.NewResolveLinks(w.db.Links()), &notes.Edit.Linking, &notes.Connect.CreateInRelation, &notes.Refer.Backlinks,
	)
	connections := notewindow.NewConnections(
		state, state, &notes.Browse.Connection, &notes.Connect.SaveConnection, &notes.Connect.RemoveSide, &notes.Connect.RemoveConnection, &notes.Connect.Join,
	)
	handler := notewindow.NewHandler(state, prose, outline, links, connections)
	w.inst.Mount(editor.NewNoteRoute(state, handler))
}
