package file

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// ImportFile writes a stream as the file at a path, whole or not at all.
type ImportFile struct {
	writers vaultport.FolderWriters
}

func NewImportFile(writers vaultport.FolderWriters) *ImportFile {
	return &ImportFile{writers: writers}
}

// Execute creates the file, or replaces the one at the path when isReplacing is set.
func (u *ImportFile) Execute(ctx context.Context, v domain.Vault, path string, content io.Reader, isReplacing bool) error {
	folders, err := u.writers.OpenFolderWriter(v)
	if err != nil {
		return err
	}
	if isReplacing {
		return folders.Replace(ctx, path, content)
	}
	return folders.Import(ctx, path, content)
}
