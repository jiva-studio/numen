package note

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

func countMatchingFiles(
	ctx context.Context, readers vaultport.VaultReaders, v domain.Vault, path string, baseline *domain.FileStat,
) (int, error) {
	reader, err := readers.Open(v)
	if err != nil {
		return 0, err
	}
	count, err := countFiles(ctx, reader, path)
	if err != nil {
		return 0, err
	}
	if count != 1 || baseline == nil {
		return count, nil
	}
	hash, err := hashFile(ctx, reader, path)
	if errors.Is(err, edit.ErrNotExist) {
		return 0, fmt.Errorf("%s is gone since it was read: %w", path, edit.ErrStale)
	}
	if err != nil {
		return 0, err
	}
	if hash != baseline.Hash {
		return 0, fmt.Errorf("%s changed since it was read: %w", path, edit.ErrStale)
	}
	return count, nil
}

// countFiles counts one for a file and the files under it for a folder.
func countFiles(ctx context.Context, reader vaultport.VaultReader, path string) (int, error) {
	if _, err := reader.StatFile(ctx, path); err == nil {
		return 1, nil
	} else if !errors.Is(err, edit.ErrNotExist) {
		return 0, err
	}
	count := 0
	err := reader.WalkFiles(ctx, func(stat domain.FileStat) error {
		if strings.HasPrefix(stat.Path, path+"/") {
			count++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	if count == 0 {
		return 0, fmt.Errorf("%s: %w", path, edit.ErrNotExist)
	}
	return count, nil
}

func hashFile(ctx context.Context, reader vaultport.VaultReader, path string) (_ string, err error) {
	if _, err := reader.StatFile(ctx, path); err != nil {
		return "", err
	}
	file, err := reader.Open(ctx, path)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	return domain.HashReader(file)
}

// FileMove moves a file or a folder inside the vault.
type FileMove interface {
	Execute(ctx context.Context, v domain.Vault, from, to string) (domain.MoveResult, bool, error)
}

// MoveChecked moves a file, or a folder with everything under it, once the file matches the baseline the caller
// read, and counts the files that moved.
type MoveChecked struct {
	move    FileMove
	readers vaultport.VaultReaders
}

func NewMoveChecked(move FileMove, readers vaultport.VaultReaders) MoveChecked {
	return MoveChecked{move: move, readers: readers}
}

// Execute returns edit.ErrStale for a file that changed since the baseline, and edit.ErrNotExist for a path with
// no file.
func (u MoveChecked) Execute(ctx context.Context, v domain.Vault, from, to string, baseline *domain.FileStat) (int, error) {
	count, err := countMatchingFiles(ctx, u.readers, v, from, baseline)
	if err != nil {
		return 0, err
	}
	_, _, err = u.move.Execute(ctx, v, from, to)
	if errors.Is(err, edit.ErrNoNote) {
		err = fmt.Errorf("%s: %w", from, edit.ErrNotExist)
	}
	if err != nil {
		return 0, err
	}
	return count, nil
}

// RemoveChecked moves a file, or a folder with everything under it, to the trash once the file matches the
// baseline the caller read, and counts the files that went.
type RemoveChecked struct {
	remove  RemoveFile
	readers vaultport.VaultReaders
}

func NewRemoveChecked(remove RemoveFile, readers vaultport.VaultReaders) RemoveChecked {
	return RemoveChecked{remove: remove, readers: readers}
}

// Execute returns edit.ErrStale for a file that changed since the baseline, and edit.ErrNotExist for a path with
// no file.
func (u RemoveChecked) Execute(ctx context.Context, v domain.Vault, path string, baseline *domain.FileStat) (int, error) {
	count, err := countMatchingFiles(ctx, u.readers, v, path, baseline)
	if err != nil {
		return 0, err
	}
	if _, _, err = u.remove.Execute(ctx, v, path); err != nil {
		return 0, err
	}
	return count, nil
}
