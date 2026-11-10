package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	windowtext "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/text"
)

func mountText(w *windowWiring) {
	handler := windowtext.NewHandler(w.inst.State, nil)
	if text := openFileText(w.cfg.vault()); text != nil {
		handler = windowtext.NewHandler(w.inst.State, text)
	}
	path, served := windowtext.NewRoute(handler, w.inst.State.Gate.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
}
