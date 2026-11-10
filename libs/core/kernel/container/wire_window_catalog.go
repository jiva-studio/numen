package container

import (
	windowcatalog "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/catalog"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
)

func mountCatalog(w *windowWiring) {
	if plugins := w.cfg.GetPluginCatalog(); plugins != nil {
		handler := windowcatalog.NewHandler(plugins)
		path, served := windowcatalog.NewRoute(handler, w.inst.State.Gate.CountQuestions())
		w.inst.Mount(editor.NewRoute(path, served, handler))
	}
}
