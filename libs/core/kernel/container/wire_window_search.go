package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	editorsearch "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/search"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
)

func mountSearch(w *windowWiring) {
	state := w.inst.State
	handler := editorsearch.NewHandler(state)
	handler.GetNameMatches = search.NewGetNameMatches(w.db.NameMatches())
	handler.GetTitles = query.NewGetTitles(w.db.NoteRefs())
	handler.GetTypes = query.NewGetTypes(w.db.NoteTypes())
	handler.ListTabKinds = w.files.ListTabKinds
	finds := w.cfg.NewSearch(w.db, w.inst.Asking, state.ReportWordsAlone)
	handler.Finds = &finds
	w.cfg.SetPluginSearch(w.db, w.inst.Embedder, state.ReportFloorError)
	w.inst.Mount(editor.NewSearchRoute(state, handler))
}
