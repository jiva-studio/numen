package editor

import (
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
)

// NewFileServer is the server of the files over the loopback socket.
func NewFileServer(
	state *State, find *registry.Find, open *file.OpenFile, plugin *file.OpenPluginFile, types *file.GetMediaType,
) *FileServer {
	return &FileServer{state: state, find: find, openFile: open, openPluginFile: plugin, mediaTypes: types}
}
