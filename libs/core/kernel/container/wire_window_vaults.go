package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/vaults"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/picker"
)

func mountVaults(w *windowWiring) {
	state := w.inst.State
	list := w.vaults
	handler := vaults.NewHandler(
		state, list.Registry.List, list.Registry.Find, list.Registry.FolderCheck, list.Change.Add, list.Change.Rename, list.Change.Remove,
	)
	w.inst.OnFolderDialog(func(dialog editor.PathDialog) {
		handler.SetFolderChooser(picker.NewChooseFolder(dialog))
	})
	path, served := vaults.NewRoute(handler, state.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
}
