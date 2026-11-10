package container

import "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"

// mountWindowState mounts the service that answers the tasks of the window and its flush.
func mountWindowState(w *windowWiring) { w.inst.Mount(editor.NewWindowRoute(w.inst.State)) }
