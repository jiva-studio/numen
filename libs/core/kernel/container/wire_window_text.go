package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	editortexts "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/texts"
)

func mountText(w *windowWiring) {
	handler := editortexts.NewHandler(w.inst.State, nil)
	if text := w.cfg.vault().OpenFileText(); text != nil {
		handler = editortexts.NewHandler(w.inst.State, text)
	}
	w.inst.Mount(editor.NewTextRoute(w.inst.State, handler))
}
