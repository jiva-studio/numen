package file

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

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

// Execute replaces the file if it still matches the baseline and returns the new file stat. A file that
// was written and not indexed returns that file stat and edit.ErrIndexOutdated.
func (u WriteFile) Execute(
	ctx context.Context, v domain.Vault, path string, content []byte, baseline domain.FileStat,
) (domain.FileStat, error) {
	writer, err := u.writers.OpenNoteWriter(v)
	if err != nil {
		return domain.FileStat{}, err
	}
	stat, err := writer.Write(ctx, path, content, baseline)
	if err != nil {
		return stat, err
	}
	return stat, edit.WrapIndexOutdated(u.index(ctx, v, []string{path}), path)
}

// CreateFile makes a file of a vault that is not there and updates the index for it.
type CreateFile struct {
	writers vaultport.NoteWriters
	index   edit.IndexUpdater
}

func NewCreateFile(writers vaultport.NoteWriters, index edit.IndexUpdater) CreateFile {
	return CreateFile{writers: writers, index: index}
}

// Execute writes the file, or returns vault.ErrAlreadyExists when the path holds one. A file that was
// written and not indexed returns edit.ErrIndexOutdated.
func (u CreateFile) Execute(ctx context.Context, v domain.Vault, path string, content []byte) error {
	writer, err := u.writers.OpenNoteWriter(v)
	if err != nil {
		return err
	}
	if err := writer.Create(ctx, path, content); err != nil {
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
