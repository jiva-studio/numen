package note

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// TrashedVersions deletes the text versions that follow files into the trash.
type TrashedVersions interface {
	// DeleteUnder deletes the versions of everything under a folder, except those
	// of the paths in kept and of what lies under them.
	DeleteUnder(ctx context.Context, v domain.Vault, folder string, kept []string) error
}

// EmptyTrash deletes everything the vault's trash holds, with the text versions of it.
type EmptyTrash struct {
	writers   TrashEmptiers
	versions  TrashedVersions
	admission vaultport.WriteAdmission
}

func NewEmptyTrash(writers TrashEmptiers, versions TrashedVersions, admission vaultport.WriteAdmission) EmptyTrash {
	return EmptyTrash{writers: writers, versions: versions, admission: admission}
}

// Execute deletes the trash content permanently and returns the paths it could
// not delete. The trash is not indexed, so the index is left as it is. The
// versions of the paths it could not delete stay with them.
func (u EmptyTrash) Execute(ctx context.Context, v domain.Vault) ([]string, error) {
	if err := vaultport.CheckWrite(ctx, v); err != nil {
		return nil, err
	}
	release, err := u.admission.Begin()
	if err != nil {
		return nil, err
	}
	defer release()
	writer, err := u.writers.OpenTrashEmptier(v)
	if err != nil {
		return nil, err
	}
	undeleted, err := writer.ClearTrash(ctx)
	if err != nil {
		return undeleted, err
	}
	return undeleted, u.versions.DeleteUnder(ctx, v, TrashDir, undeleted)
}
