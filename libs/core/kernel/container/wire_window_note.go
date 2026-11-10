package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
)

func mountNote(w *windowWiring) {
	state := w.inst.State
	handler := editor.NewNoteHandler(state)
	handler.SetQueries(
		query.NewGetOpeningNote(w.db.OpeningNote()),
		query.NewGetTypes(w.db.NoteTypes()),
		query.NewGetHeadings(w.db.Headings()),
		query.NewResolveLinks(w.db.Links()),
	)
	handler.SetTabKinds(w.files.ListTabKinds)
	handler.SetScenarios(w.notes)
	w.inst.Mount(editor.NewNoteRoute(state, handler))
}
