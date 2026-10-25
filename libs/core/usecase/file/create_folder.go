package file

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// CreateFolder makes an empty folder with the folders above it.
// Only the failure to make the folder wraps its cause.
type CreateFolder struct {
	writers vaultport.VaultWriters
}

func NewCreateFolder(writers vaultport.VaultWriters) *CreateFolder {
	return &CreateFolder{writers: writers}
}

func (u *CreateFolder) Execute(ctx context.Context, v domain.Vault, path string) error {
	writer, err := u.writers.Open(v)
	if err != nil {
		return newVaultFailure(err)
	}
	return writer.MakeFolder(ctx, path)
}
