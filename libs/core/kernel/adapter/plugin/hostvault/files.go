package hostvault

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
)

// Files reads and writes the files of the vault. Locks, where set, holds the writers of the vault apart while a call writes.
type Files struct {
	Stat        *file.StatFile
	ReadFile    *file.ReadFile
	RangeReader file.Read
	Write       file.WriteFile
	Create      file.CreateFile
	Index       func(ctx context.Context, v domain.Vault, paths []string) error
	Locks       vault.Locks

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
func (f Files) WriteFile(
	ctx context.Context, path string, content []byte, baseline domain.FileStat,
) (domain.FileStat, error) {
	v, err := f.session.open()
	if err != nil {
		return domain.FileStat{}, err
	}
	stat, err := f.Write.Execute(ctx, v, path, content, baseline)
	return stat, f.session.notifyAfterWrite(err)
}

func (f Files) CreateFile(ctx context.Context, path string, content []byte) error {
	v, err := f.session.open()
	if err != nil {
		return err
	}
	return f.session.notifyAfterWrite(f.Create.Execute(ctx, v, path, content))
}
