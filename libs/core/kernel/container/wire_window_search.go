package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	windowsearch "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/search"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
)

func mountSearch(w *windowWiring) {
	state := w.inst.State
	handler := windowsearch.NewHandler(state)
	handler.GetNameMatches = search.NewGetNameMatches(w.db.NameMatches())
	handler.GetTitles = query.NewGetTitles(w.db.NoteRefs())
	handler.GetTypes = query.NewGetTypes(w.db.NoteTypes())
	handler.ListTabKinds = w.files.ListTabKinds
	finds := w.cfg.NewSearch(w.db, w.asking, state.ReportWordsAlone)
	handler.Finds = &finds
	w.cfg.setPluginSearch(w.db, w.embedder, state.ReportFloorError)
	path, served := windowsearch.NewRoute(handler, state.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
}
