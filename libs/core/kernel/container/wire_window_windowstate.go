package container

import "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"

func mountWindowState(w *windowWiring) { w.inst.Mount(editor.NewWindowRoute(w.inst.State)) }
