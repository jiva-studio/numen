package note

import (
	"context"

	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/windowport"
)

// TrashedVersions deletes the text versions that follow files into the trash.
type TrashedVersions interface {
	// DeleteUnder deletes the versions of everything under a folder, except those
	// of the paths in kept and of what lies under them.
	DeleteUnder(ctx context.Context, v domain.Vault, folder string, kept []string) error
}

// EmptyTrash deletes everything the vault's trash holds, with the text versions of it.
type EmptyTrash struct {
	writers  vaultport.TrashEmptiers
	versions TrashedVersions
	gate     windowport.WriteGate
}

func NewEmptyTrash(writers vaultport.TrashEmptiers, versions TrashedVersions, gate windowport.WriteGate) EmptyTrash {
	return EmptyTrash{writers: writers, versions: versions, gate: gate}
}

// Execute deletes the trash content permanently and returns the paths it could
// not delete. The trash is not indexed, so the index is left as it is. The
// versions of the paths it could not delete stay with them.
func (u EmptyTrash) Execute(ctx context.Context, v domain.Vault) ([]string, error) {
	if !u.gate.BeginWrite() {
		return nil, windowdomain.ErrClosing
	}
	defer u.gate.FinishWrite()
	writer, err := u.writers.OpenTrashEmptier(v)
	if err != nil {
		return nil, err
	}
	undeleted, err := writer.EmptyTrash(ctx)
	if err != nil {
		return undeleted, err
	}
	return undeleted, u.versions.DeleteUnder(ctx, v, TrashDir, undeleted)
}
