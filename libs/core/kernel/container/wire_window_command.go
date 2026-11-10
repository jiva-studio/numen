package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	editorcommand "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/command"
)

func mountCommand(w *windowWiring) {
	var versions editorcommand.VersionItems
	if menu := w.cfg.plugins().OpenVersionItems(); menu != nil {
		versions = menu
	}
	handler := editorcommand.NewHandler(w.inst.State, w.cfg.GetPluginCommands(), versions)
	w.inst.Mount(editor.NewCommandRoute(w.inst.State, handler))
}
