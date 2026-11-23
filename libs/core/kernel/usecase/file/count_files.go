package file

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// CountFiles counts the files at a path: one for a file, and the files under it for a folder.
type CountFiles struct {
	readers vaultport.VaultReaders
}

func NewCountFiles(readers vaultport.VaultReaders) *CountFiles {
	return &CountFiles{readers: readers}
}

// Execute returns fs.ErrNotExist for a path with no file and no folder holding one.
func (u *CountFiles) Execute(ctx context.Context, v domain.Vault, path string) (int, error) {
	reader, err := u.readers.Open(v)
	if err != nil {
		return 0, err
	}
	if _, err := reader.StatFile(ctx, path); err == nil {
		return 1, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return 0, err
	}
	count := 0
	err = reader.WalkFiles(ctx, func(stat domain.FileStat) error {
		if strings.HasPrefix(stat.Path, path+"/") {
			count++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	if count == 0 {
		return 0, fmt.Errorf("%s: %w", path, fs.ErrNotExist)
	}
	return count, nil
}
