package note

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/indexport"
)

// RemoveFile moves a file or a folder of the vault to the trash as a write of the window.
type RemoveFile struct {
	remove *Remove
	follow indexport.IndexFollow
}

// NewRemoveFile returns the use case over the removal of a file or a folder.
func NewRemoveFile(remove *Remove, follow indexport.IndexFollow) RemoveFile {
	return RemoveFile{remove: remove, follow: follow}
}

// Execute puts the file or folder at path in the trash and reports whether the index did not follow. It
// fails with window.ErrClosing once the window is closing.
func (u RemoveFile) Execute(ctx context.Context, v domain.Vault, path string) (RemoveResult, bool, error) {
	removed, err := u.remove.Execute(ctx, v, path)
	return removed, u.follow.IsIndexOutdated(err), err
}
