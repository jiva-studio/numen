package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/windowstate"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/workspace"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
)

func mountWindowState(w *windowWiring) {
	state := w.inst.State
	path, served := windowstate.NewRoute(state.Window, state.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, state.Window))
}

func mountWorkspace(w *windowWiring) error {
	state := w.inst.State
	links, err := workspace.NewLinks()
	if err != nil {
		return err
	}
	store := w.db.Stores.OpenTabs
	handler := workspace.NewHandler(
		state, state, vault.NewWriteOpenTabs(store), vault.NewReadOpenTabs(store), links,
	)
	path, served := workspace.NewRoute(handler, state.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
	return nil
}
