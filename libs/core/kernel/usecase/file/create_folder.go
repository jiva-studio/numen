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

type openWriteGate struct{}

func (openWriteGate) BeginWrite() bool { return true }
func (openWriteGate) FinishWrite()     {}

// CreateFolder makes an empty folder with the folders above it.
// Only the failure to make the folder wraps its cause.
type CreateFolder struct {
	writers vaultport.FolderWriters
	gate    WriteGate
}

func NewCreateFolder(writers vaultport.FolderWriters) *CreateFolder {
	return &CreateFolder{writers: writers, gate: openWriteGate{}}
}

// WithWriteGate returns CreateFolder gated by gate.
func (u *CreateFolder) WithWriteGate(gate WriteGate) *CreateFolder {
	return &CreateFolder{writers: u.writers, gate: gate}
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
