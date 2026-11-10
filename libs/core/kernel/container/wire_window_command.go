package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/command"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
)

func mountCommand(w *windowWiring) {
	var versions command.VersionItems
	if menu := w.cfg.plugins().OpenVersionItems(); menu != nil {
		versions = menu
	}
	handler := command.NewHandler(w.inst.State, w.cfg.GetPluginCommands(), versions)
	w.inst.Mount(editor.NewRoute(command.NewRoute(w.inst.State.CountQuestions(), handler)))
}
