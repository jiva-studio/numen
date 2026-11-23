package note

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"
)

// IndexFollow reports whether the index followed a write.
type IndexFollow interface {
	// IsIndexOutdated reports whether err says the write reached the vault and the index did not follow.
	IsIndexOutdated(err error) bool
}

// SourcesUnder answers which sources the index holds at a path and beneath it.
type SourcesUnder interface {
	GetSourcesUnder(ctx context.Context, vaultID domain.VaultID, path string) ([]domain.FileStat, error)
}

// FileMovers opens a vault for moving and removing its files.
type FileMovers interface {
	OpenFileMover(v domain.Vault) (vaultport.FileMover, error)
}

// isIndexOutdated reports whether err says the write reached the vault and the index did not follow. Without a
// follow it reads the error itself.
func isIndexOutdated(follow IndexFollow, err error) bool {
	if follow == nil {
		return errors.Is(err, edit.ErrIndexOutdated)
	}
	return follow.IsIndexOutdated(err)
}
