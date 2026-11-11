package hostvault

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
)

// NoteWriter writes and creates the notes of one vault.
type NoteWriter interface {
	Write(ctx context.Context, path string, content []byte, stat domain.FileStat) (domain.FileStat, error)
	Create(ctx context.Context, path string, content []byte) error
}

// Files reads and writes the files of the vault.
type Files struct {
	Stat        *file.StatFile
	ReadFile    *file.ReadFile
	RangeReader file.Read
	OpenWriter  func(v domain.Vault) (NoteWriter, error)
	Index       func(ctx context.Context, v domain.Vault, paths []string) error

	session *Session
}

func (f Files) openWriter() (NoteWriter, error) {
	v, err := f.session.open()
	if err != nil {
		return nil, err
	}
	return f.OpenWriter(v)
}

// StatFile returns the stat of the file at the path.
func (f Files) StatFile(ctx context.Context, path string) (domain.FileStat, error) {
	v, err := f.session.open()
	if err != nil {
		return domain.FileStat{}, err
	}
	return f.Stat.Execute(ctx, v, path)
}

// Read returns the bytes of the file and its stat.
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
	writer, err := f.openWriter()
	if err != nil {
		return domain.FileStat{}, err
	}
	pos, err := writer.Write(ctx, path, content, baseline)
	if err != nil {
		return pos, err
	}
	if err := f.Scan(ctx, []string{path}); err != nil {
		return pos, fmt.Errorf("index %s after write: %w", path, err)
	}
	return pos, nil
}

// CreateFile creates the file and indexes it.
func (f Files) CreateFile(ctx context.Context, path string, content []byte) error {
	writer, err := f.openWriter()
	if err != nil {
		return err
	}
	if err := writer.Create(ctx, path, content); err != nil {
		return err
	}
	if err := f.Scan(ctx, []string{path}); err != nil {
		return fmt.Errorf("index %s after create: %w", path, err)
	}
	return nil
}

// Scan updates the index for the files named.
func (f Files) Scan(ctx context.Context, paths []string) error {
	v, err := f.session.open()
	if err != nil {
		return err
	}
	f.session.notifyFileWritten()
	if f.Index == nil {
		return nil
	}
	return f.Index(ctx, v, paths)
}
