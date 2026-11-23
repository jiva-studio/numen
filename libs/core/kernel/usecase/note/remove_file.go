package note

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// RemoveFile moves a file or a folder of the vault to the trash.
type RemoveFile struct {
	remove *Remove
	follow IndexFollow
}

// NewRemoveFile returns the use case over the removal of a file or a folder.
func NewRemoveFile(remove *Remove, follow IndexFollow) RemoveFile {
	return RemoveFile{remove: remove, follow: follow}
}

// Execute puts the file or folder at path in the trash and reports whether the index did not follow.
func (u RemoveFile) Execute(ctx context.Context, v domain.Vault, path string) (RemoveResult, bool, error) {
	removed, err := u.remove.Execute(ctx, v, path)
	return removed, isIndexOutdated(u.follow, err), err
}
