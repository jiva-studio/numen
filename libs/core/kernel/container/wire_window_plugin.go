package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/plugincatalog"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
)

func mountPlugin(w *windowWiring) {
	w.inst.SetPlugins(w.cfg.GetPluginServices(), w.cfg.GetPluginDir)
	w.inst.Mount(editor.NewPluginRoute(w.inst.State))
}

func mountCatalog(w *windowWiring) {
	if plugins := w.cfg.getEditorCatalog(); plugins != nil {
		w.inst.Mount(editor.NewCatalogRoute(w.inst.State, plugincatalog.NewHandler(plugins)))
	}
}

// SetEditorPluginWindow gives the plugins the window: it focuses a note and the passages of a file
// in it and returns the tabs it last reported. The window reads the vault of its state on each call,
// so a vault opened later in the window is served without a new binding.
func (c Config) SetEditorPluginWindow(db *Index, opened *editor.Installation) {
	view := opened.State.GetWindow()
	c.SetPluginWindow(editor.NewPluginWindow(
		opened.State, c.FocusNote(db.NoteRefs(), view), file.NewFocusPassages(c.VaultReaders(), view),
	))
}
