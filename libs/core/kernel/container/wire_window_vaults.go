package container

import "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"

// mountVaults mounts the service that lists and changes the vaults of the installation.
func (w *windowWiring) mountVaults() {
	handler := editor.NewVaultsHandler(w.inst.State)
	handler.SetScenarios(w.vaults)
	w.inst.OnFolderDialog(handler.SetFolderDialog)
	w.inst.Mount(editor.NewVaultsRoute(w.inst.State, handler))
}
