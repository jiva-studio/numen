package note

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"
)

// FileCounter counts the files at a path: one for a file, and the files under it for a folder.
type FileCounter func(ctx context.Context, v domain.Vault, path string) (int, error)

// FileHasher returns the stat of a file with the hash of its content.
type FileHasher func(ctx context.Context, v domain.Vault, path string) (domain.FileStat, error)

func countMatchingFiles(
	ctx context.Context, files FileCounter, hashes FileHasher, v domain.Vault, path string, baseline *domain.FileStat,
) (int, error) {
	count, err := files(ctx, v, path)
	if err != nil {
		return 0, err
	}
	if count != 1 || baseline == nil {
		return count, nil
	}
	now, err := hashes(ctx, v, path)
	if errors.Is(err, edit.ErrNotExist) {
		return 0, fmt.Errorf("%s is gone since it was read: %w", path, edit.ErrStale)
	}
	if err != nil {
		return 0, err
	}
	if now.Hash != baseline.Hash {
		return 0, fmt.Errorf("%s changed since it was read: %w", path, edit.ErrStale)
	}
	return count, nil
}

// FileMove moves a file or a folder inside the vault.
type FileMove interface {
	Execute(ctx context.Context, v domain.Vault, from, to string) (domain.MoveResult, bool, error)
}

// MoveChecked moves a file, or a folder with everything under it, once the file matches the baseline the caller
// read, and counts the files that moved.
type MoveChecked struct {
	move   FileMove
	files  FileCounter
	hashes FileHasher
}

func NewMoveChecked(move FileMove, files FileCounter, hashes FileHasher) MoveChecked {
	return MoveChecked{move: move, files: files, hashes: hashes}
}

// Execute returns edit.ErrStale for a file that changed since the baseline, and edit.ErrNotExist for a path with
// no file.
func (u MoveChecked) Execute(ctx context.Context, v domain.Vault, from, to string, baseline *domain.FileStat) (int, error) {
	count, err := countMatchingFiles(ctx, u.files, u.hashes, v, from, baseline)
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
	remove RemoveFile
	files  FileCounter
	hashes FileHasher
}

func NewRemoveChecked(remove RemoveFile, files FileCounter, hashes FileHasher) RemoveChecked {
	return RemoveChecked{remove: remove, files: files, hashes: hashes}
}

// Execute returns edit.ErrStale for a file that changed since the baseline, and edit.ErrNotExist for a path with
// no file.
func (u RemoveChecked) Execute(ctx context.Context, v domain.Vault, path string, baseline *domain.FileStat) (int, error) {
	count, err := countMatchingFiles(ctx, u.files, u.hashes, v, path, baseline)
	if err != nil {
		return 0, err
	}
	if _, _, err = u.remove.Execute(ctx, v, path); err != nil {
		return 0, err
	}
	return count, nil
}
