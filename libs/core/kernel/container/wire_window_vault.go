package container

import "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"

// mountVault mounts the service that answers what the vault in the window is doing.
func (w *windowWiring) mountVault() { w.inst.Mount(editor.NewVaultRoute(w.inst.State)) }
