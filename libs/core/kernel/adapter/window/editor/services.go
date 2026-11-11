package editor

import (
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
)

// NewFileServer is the server of the files over the loopback socket.
func NewFileServer(state *State, open *file.OpenVaultFile) *FileServer {
	return &FileServer{state: state, open: open}
}
