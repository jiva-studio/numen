package file

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// ReadFile returns the whole content of the file at a path, with what the vault reports about the
// file as it was before the bytes were read.
type ReadFile struct {
	readers vaultport.VaultReaders
}

func NewReadFile(readers vaultport.VaultReaders) *ReadFile {
	return &ReadFile{readers: readers}
}

func (u *ReadFile) Execute(ctx context.Context, v domain.Vault, path string) ([]byte, domain.FileStat, error) {
	reader, err := u.readers.Open(v)
	if err != nil {
		return nil, domain.FileStat{}, err
	}
	stat, err := reader.StatFile(ctx, path)
	if err != nil {
		return nil, domain.FileStat{}, err
	}
	content, err := reader.Read(ctx, path)
	if err != nil {
		return nil, domain.FileStat{}, err
	}
	return content, stat, nil
}
