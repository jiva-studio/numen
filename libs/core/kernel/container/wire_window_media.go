package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	mediawindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/media"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
)

func mountMedia(w *windowWiring) {
	state := w.inst.State
	resolve := file.NewResolveAddress(file.NewStatFile(w.cfg.VaultReaders()), openPluginFile(w.cfg.vault()))
	handler := mediawindow.NewHandler(state, state, resolve, getMediaType(w.cfg.vault()))
	w.inst.Mount(editor.NewMediaRoute(state, handler))
}
