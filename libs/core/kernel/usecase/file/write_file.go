package file

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// WriteFile replaces a file of a vault and updates the index for it.
type WriteFile struct {
	writers vaultport.NoteWriters
	index   edit.IndexUpdater
}

func NewWriteFile(writers vaultport.NoteWriters, index edit.IndexUpdater) WriteFile {
	return WriteFile{writers: writers, index: index}
}

// Execute replaces the file if it still matches the baseline and returns the new file stat, with its
// hash and the version of the vault. A file that was written and not indexed returns that file stat
// and edit.ErrIndexOutdated.
func (u WriteFile) Execute(
	ctx context.Context, v domain.Vault, path string, content []byte, baseline domain.FileStat,
) (domain.FileStat, error) {
	if err := vaultport.CheckWrite(ctx, v); err != nil {
		return domain.FileStat{}, err
	}
	writer, err := u.writers.OpenNoteWriter(v)
	if err != nil {
		return domain.FileStat{}, err
	}
	written, err := writer.Write(ctx, path, content, baseline)
	if err != nil {
		return domain.FileStat{}, err
	}
	stat := written.Stat
	stat.Version = written.Version
	return stat, edit.WrapIndexOutdated(u.index(ctx, v, []string{path}), path)
}

// FileCreator creates a new vault file with the supplied bytes.
type FileCreator interface {
	CreateFile(ctx context.Context, v domain.Vault, path string, content []byte) error
}

// CreateFile makes a file of a vault that is not there and updates the index for it.
type CreateFile struct {
	writers FileCreator
	index   edit.IndexUpdater
}

func NewCreateFile(writers FileCreator, index edit.IndexUpdater) CreateFile {
	return CreateFile{writers: writers, index: index}
}

// Execute writes the file, or returns vault.ErrAlreadyExists when the path holds one. A file that was
// written and not indexed returns edit.ErrIndexOutdated.
func (u CreateFile) Execute(ctx context.Context, v domain.Vault, path string, content []byte) error {
	if err := u.writers.CreateFile(ctx, v, path, content); err != nil {
		return err
	}
	return edit.WrapIndexOutdated(u.index(ctx, v, []string{path}), path)
}

// ScanFiles updates the index for files of a vault that were written outside the core.
type ScanFiles struct {
	index edit.IndexUpdater
}

func NewScanFiles(index edit.IndexUpdater) ScanFiles {
	return ScanFiles{index: index}
}

// Execute indexes the files at the paths. A failed index update returns edit.ErrIndexOutdated.
func (u ScanFiles) Execute(ctx context.Context, v domain.Vault, paths []string) error {
	return edit.WrapIndexOutdated(u.index(ctx, v, paths), paths...)
}
