package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/windowstate"
)

func mountWindowState(w *windowWiring) {
	state := w.inst.State
	path, served := windowstate.NewRoute(state.Window, state.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, state.Window))
}
