package note

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// EmptyTrash deletes everything the vault's trash holds.
type EmptyTrash struct {
	writers vaultport.VaultWriters
}

func NewEmptyTrash(writers vaultport.VaultWriters) EmptyTrash {
	return EmptyTrash{writers: writers}
}

// Execute deletes the trash content permanently and returns the paths it could
// not delete. The trash is not indexed, so the index is left as it is.
func (u EmptyTrash) Execute(ctx context.Context, v domain.Vault) ([]string, error) {
	writer, err := u.writers.Open(v)
	if err != nil {
		return nil, err
	}
	return writer.EmptyTrash(ctx)
}
