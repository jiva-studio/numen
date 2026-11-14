package hostvault

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
)

// Files reads the files of the vault.
type Files struct {
	Stat        *file.StatFile
	ReadFile    *file.ReadFile
	RangeReader file.Read

	session sessionAccess
}

// Writer writes the files of the vault and indexes them. Locks, where set, holds the writers of the vault
// apart while a call writes.
type Writer struct {
	Write  file.WriteFile
	Create file.CreateFile
	Index  file.ScanFiles
	Locks  vault.Locks

	session sessionAccess
}

func (f Files) StatFile(ctx context.Context, path string) (domain.FileStat, error) {
	v, err := f.session.open()
	if err != nil {
		return domain.FileStat{}, err
	}
	return f.Stat.Execute(ctx, v, path)
}

func (f Files) Read(ctx context.Context, path string) ([]byte, domain.FileStat, error) {
	v, err := f.session.open()
	if err != nil {
		return nil, domain.FileStat{}, err
	}
	return f.ReadFile.Execute(ctx, v, path)
}

// ReadRange reads the range through the read of usecase/file, with the file stat taken before the
// bytes are read.
func (f Files) ReadRange(ctx context.Context, path string, start, length int) (plugin.FileRange, error) {
	v, err := f.session.open()
	if err != nil {
		return plugin.FileRange{}, err
	}
	return f.RangeReader.ExecuteFileRange(ctx, v, path, start, length)
}

// WriteFile replaces the file if it still matches the baseline, and indexes it.
func (w Writer) WriteFile(
	ctx context.Context, path string, content []byte, baseline domain.FileStat,
) (domain.FileStat, error) {
	v, err := w.session.open()
	if err != nil {
		return domain.FileStat{}, err
	}
	stat, err := w.Write.Execute(ctx, v, path, content, baseline)
	return stat, w.session.notifyAfterWrite(err)
}

func (w Writer) CreateFile(ctx context.Context, path string, content []byte) error {
	v, err := w.session.open()
	if err != nil {
		return err
	}
	return w.session.notifyAfterWrite(w.Create.Execute(ctx, v, path, content))
}

// ScanFiles updates the index for the files named, which the plugin wrote outside the vault calls.
func (w Writer) ScanFiles(ctx context.Context, paths []string) error {
	v, err := w.session.open()
	if err != nil {
		return err
	}
	return w.session.notifyAfterWrite(w.Index.Execute(ctx, v, paths))
}
