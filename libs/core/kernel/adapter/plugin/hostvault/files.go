package hostvault

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
)

// Files reads the files of the vault.
type Files struct {
	Stat        *file.HashFile
	ReadFile    *file.ReadFile
	RangeReader file.Read
	Bytes       *file.ReadBytes

	session sessionAccess
}

// fileStatter is the stat of a file with its hash, which the use case file.HashFile gives.
type fileStatter interface {
	Execute(ctx context.Context, v domain.Vault, path string) (domain.FileStat, error)
}

// fileCounter counts the files at a path, which the use case file.CountFiles does.
type fileCounter interface {
	Execute(ctx context.Context, v domain.Vault, path string) (int, error)
}

// fileImporter writes a stream as the file at a path, which the use case file.ImportFile does.
type fileImporter interface {
	Execute(ctx context.Context, v domain.Vault, path string, content io.Reader, isReplacing bool) error
}

// Writer writes the files of the vault and indexes them. Locks, where set, holds the writers of the vault
// apart while a call writes.
type Writer struct {
	Write  file.WriteFile
	Create file.CreateFile
	Index  file.ScanFiles
	Locks  vault.Locks

	// The files of any kind: moved and removed as the window does, and placed from the store folder of a plugin.
	Move         note.MoveFile
	Remove       note.RemoveFile
	Stat         fileStatter
	Count        fileCounter
	Import       fileImporter
	StoreReaders func(v domain.Vault, pluginID string) (vault.StoreReader, error)
	StoreWriters func(v domain.Vault, pluginID string) (vault.StoreWriter, error)

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
	w.session.ReportWrites(v.ID, err)
	return stat, err
}

func (w Writer) CreateFile(ctx context.Context, path string, content []byte) error {
	v, err := w.session.open()
	if err != nil {
		return err
	}
	err = w.Create.Execute(ctx, v, path, content)
	w.session.ReportWrites(v.ID, err)
	return err
}

// ScanFiles updates the index for the files named, which the plugin wrote outside the vault calls.
func (w Writer) ScanFiles(ctx context.Context, paths []string) error {
	v, err := w.session.open()
	if err != nil {
		return err
	}
	err = w.Index.Execute(ctx, v, paths)
	w.session.ReportWrites(v.ID, err)
	return err
}

// ReadFileRange reads bytes of any file as they are, with the file stat taken before them.
func (f Files) ReadFileRange(ctx context.Context, path string, start, length int64) ([]byte, domain.FileStat, error) {
	v, err := f.session.open()
	if err != nil {
		return nil, domain.FileStat{}, err
	}
	return f.Bytes.Execute(ctx, v, path, start, length)
}
