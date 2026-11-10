package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/plugincatalog"
)

// mountCatalog mounts the service that lists the plugins the application has. A build put together
// without a catalog serves none.
func mountCatalog(w *windowWiring) {
	if plugins := w.cfg.getEditorCatalog(); plugins != nil {
		w.inst.Mount(editor.NewCatalogRoute(w.inst.State, plugincatalog.NewHandler(plugins)))
	}
}
