package editor

import (
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
)

// NewFileServer is the server of the files over the loopback socket.
func NewFileServer(vaults shownVault, tasks taskList, open *file.OpenVaultFile) *FileServer {
	return &FileServer{vaults: vaults, tasks: tasks, open: open}
}
