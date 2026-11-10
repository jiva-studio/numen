package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	editortexts "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/texts"
)

// mountText mounts the service that reads the text of the files of the vault. Without text versions
// it answers with none.
func (w *windowWiring) mountText() {
	handler := editortexts.NewHandler(w.inst.State, nil)
	if text := w.cfg.vault().OpenFileText(); text != nil {
		handler = editortexts.NewHandler(w.inst.State, text)
	}
	w.inst.Mount(editor.NewTextRoute(w.inst.State, handler))
}
