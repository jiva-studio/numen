package container

import "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"

// mountWindowState mounts the service that answers the tasks of the window and its flush.
func (w *windowWiring) mountWindowState() { w.inst.Mount(editor.NewWindowRoute(w.inst.State)) }
