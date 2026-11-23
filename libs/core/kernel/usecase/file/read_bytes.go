package file

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// MostBytes is how many bytes of a file one read of bytes carries.
const MostBytes = 4 << 20

type ReadBytes struct {
	readers vaultport.VaultReaders
	stat    *HashFile
}

func NewReadBytes(readers vaultport.VaultReaders, stat *HashFile) *ReadBytes {
	return &ReadBytes{readers: readers, stat: stat}
}

// Execute returns at most length bytes of the file from start, and the file stat, with its hash, taken before
// they are read. A start below zero and a length outside 1 to MostBytes return plugin.ErrInvalidRange. A start
// at or past the end returns no bytes. A folder returns fs.ErrNotExist.
func (u *ReadBytes) Execute(ctx context.Context, v domain.Vault, path string, start, length int64) (_ []byte, _ domain.FileStat, err error) {
	if start < 0 || length < 1 || length > MostBytes {
		return nil, domain.FileStat{}, fmt.Errorf("read %d bytes of %s from %d: %w", length, path, start, plugin.ErrInvalidRange)
	}
	stat, err := u.stat.ExecuteKnown(ctx, v, path)
	if err != nil {
		return nil, domain.FileStat{}, err
	}
	if start >= stat.Size {
		return []byte{}, stat, nil
	}
	reader, err := u.readers.Open(v)
	if err != nil {
		return nil, domain.FileStat{}, err
	}
	file, err := reader.Open(ctx, path)
	if err != nil {
		return nil, domain.FileStat{}, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	if _, err := file.Seek(start, io.SeekStart); err != nil {
		return nil, domain.FileStat{}, fmt.Errorf("seek %s: %w", path, err)
	}
	content, err := io.ReadAll(io.LimitReader(file, length))
	if err != nil {
		return nil, domain.FileStat{}, fmt.Errorf("read %s: %w", path, err)
	}
	return content, stat, nil
}
