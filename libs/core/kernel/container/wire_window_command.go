package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	editorcommand "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/command"
)

// mountCommand mounts the service that lists and runs the commands of the plugins.
func (w *windowWiring) mountCommand() {
	var versions editorcommand.VersionItems
	if menu := w.cfg.plugins().OpenVersionItems(); menu != nil {
		versions = menu
	}
	handler := editorcommand.NewHandler(w.inst.State, w.cfg.GetPluginCommands(), versions)
	w.inst.Mount(editor.NewCommandRoute(w.inst.State, handler))
}
