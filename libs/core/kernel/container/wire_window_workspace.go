package container

import "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"

func mountWorkspace(w *windowWiring) { w.inst.Mount(editor.NewWorkspaceRoute(w.inst.State)) }
