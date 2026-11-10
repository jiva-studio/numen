package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/plugincatalog"
)

func mountCatalog(w *windowWiring) {
	if plugins := w.cfg.getEditorCatalog(); plugins != nil {
		w.inst.Mount(editor.NewCatalogRoute(w.inst.State, plugincatalog.NewHandler(plugins)))
	}
}
