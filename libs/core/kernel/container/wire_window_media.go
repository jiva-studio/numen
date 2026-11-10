package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
)

// mountMedia mounts the service that tells where a media file is played from.
func mountMedia(w *windowWiring) {
	state := w.inst.State
	resolve := file.NewResolveAddress(file.NewStatFile(w.cfg.VaultReaders()), w.cfg.vault().OpenPluginFile())
	handler := editor.NewMediaHandler(state, resolve, w.cfg.vault().GetMediaType())
	w.inst.Mount(editor.NewMediaRoute(state, handler))
}
