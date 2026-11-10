package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/vaults"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/picker"
)

func mountVaults(w *windowWiring) {
	state := w.inst.State
	list := w.vaults
	handler := vaults.NewHandler(
		state, list.List, list.Find, list.FolderCheck, list.Add, list.Rename, list.Remove,
	)
	w.inst.OnFolderDialog(func(dialog machineport.PathDialog) {
		handler.SetFolderChooser(picker.NewChooseFolder(dialog))
	})
	path, served := vaults.NewRoute(handler, state.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
}
