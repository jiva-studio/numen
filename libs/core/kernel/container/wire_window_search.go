package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	editorsearch "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/search"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
)

// mountSearch mounts the search service. The window's search is the application's search. It is
// built after the embedder is resolved; when the model could not be loaded, search uses words only.
// The degraded state is reported as a task because a window opened from a desktop entry has no
// terminal.
func (w *windowWiring) mountSearch() {
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
