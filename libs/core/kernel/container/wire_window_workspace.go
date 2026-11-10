package container

import "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"

// mountWorkspace mounts the service that answers the tabs and the places of the window.
func (w *windowWiring) mountWorkspace() { w.inst.Mount(editor.NewWorkspaceRoute(w.inst.State)) }
