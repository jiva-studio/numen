package file

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// StatFile returns what the vault reports about the file at a path: its kind,
// its size and its modification time.
type StatFile struct {
	readers vaultport.VaultReaders
}

func NewStatFile(readers vaultport.VaultReaders) *StatFile {
	return &StatFile{readers: readers}
}

// Execute reads no bytes of the file.
func (u *StatFile) Execute(ctx context.Context, v domain.Vault, path string) (domain.FileStat, error) {
	reader, err := u.readers.Open(v)
	if err != nil {
		return domain.FileStat{}, err
	}
	return reader.StatFile(ctx, path)
}
