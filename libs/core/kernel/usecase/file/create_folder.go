package file

import (
	"context"

	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// WriteGate admits the writes a window waits for before it closes.
type WriteGate interface {
	BeginWrite() bool
	FinishWrite()
}

// OpenGate admits every write, for an installation with no window that waits for writes.
type OpenGate struct{}

func (OpenGate) BeginWrite() bool { return true }
func (OpenGate) FinishWrite()     {}

// CreateFolder makes an empty folder with the folders above it.
// Only the failure to make the folder wraps its cause.
type CreateFolder struct {
	writers vaultport.FolderWriters
	gate    WriteGate
}

func NewCreateFolder(writers vaultport.FolderWriters, gate WriteGate) *CreateFolder {
	return &CreateFolder{writers: writers, gate: gate}
}

func (u *CreateFolder) Execute(ctx context.Context, v domain.Vault, path string) error {
	if !u.gate.BeginWrite() {
		return windowdomain.ErrClosing
	}
	defer u.gate.FinishWrite()
	writer, err := u.writers.OpenFolderWriter(v)
	if err != nil {
		return newVaultFailure(err)
	}
	return writer.MakeFolder(ctx, path)
}
