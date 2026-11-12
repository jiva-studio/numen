package note

import (
	"context"

	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// EmptyTrash deletes everything the vault's trash holds.
type EmptyTrash struct {
	writers vaultport.TrashEmptiers
	gate    WriteGate
}

func NewEmptyTrash(writers vaultport.TrashEmptiers, gate WriteGate) EmptyTrash {
	return EmptyTrash{writers: writers, gate: gate}
}

// Execute deletes the trash content permanently and returns the paths it could
// not delete. The trash is not indexed, so the index is left as it is.
func (u EmptyTrash) Execute(ctx context.Context, v domain.Vault) ([]string, error) {
	if !u.gate.BeginWrite() {
		return nil, windowdomain.ErrClosing
	}
	defer u.gate.FinishWrite()
	writer, err := u.writers.OpenTrashEmptier(v)
	if err != nil {
		return nil, err
	}
	return writer.EmptyTrash(ctx)
}
