package file

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// CreateFolder makes an empty folder with the folders above it.
// Only the failure to make the folder wraps its cause.
type CreateFolder struct {
	writers port.VaultWriters
}

func NewCreateFolder(writers port.VaultWriters) *CreateFolder {
	return &CreateFolder{writers: writers}
}

func (u *CreateFolder) Execute(ctx context.Context, v domain.Vault, path string) error {
	writer, err := u.writers.Open(v)
	if err != nil {
		return newVaultFailure(err)
	}
	return writer.MakeFolder(ctx, path)
}
