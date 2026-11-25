package file

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// PlaceStagedFile puts a staged file in the vault at a path: the content goes to a temporary file next to the target and
// is renamed over it, so no reader sees a file that is not whole.
type PlaceStagedFile struct {
	stat    *HashFile
	scan    ScanFiles
	writers vaultport.FolderWriters
}

func NewPlaceStaged(stat *HashFile, scan ScanFiles, writers vaultport.FolderWriters) *PlaceStagedFile {
	return &PlaceStagedFile{stat: stat, scan: scan, writers: writers}
}

// PlaceInput is the file to place: its content, the hash the content must have, the path it goes to, and the file
// that path held when the caller last read it (nil for a path that must be free).
type PlaceInput struct {
	Path         string
	Staged       io.ReadSeeker
	ExpectedHash string
	Baseline     *domain.FileStat
}

// Execute returns plugin.ErrHashMismatch for a content that does not have the expected hash, edit.ErrStale for a
// file that changed since the baseline, and vault.ErrAlreadyExists for a path that is not free.
func (u *PlaceStagedFile) Execute(ctx context.Context, v domain.Vault, in PlaceInput) (domain.FileStat, error) {
	if err := verifyStagedHash(in.Staged, in.ExpectedHash); err != nil {
		return domain.FileStat{}, err
	}
	if err := u.checkTarget(ctx, v, in.Path, in.Baseline); err != nil {
		return domain.FileStat{}, err
	}
	folders, err := u.writers.OpenFolderWriter(v)
	if err != nil {
		return domain.FileStat{}, err
	}
	if in.Baseline != nil {
		err = folders.Replace(ctx, in.Path, in.Staged, *in.Baseline)
	} else {
		err = folders.Import(ctx, in.Path, in.Staged)
	}
	if err != nil {
		return domain.FileStat{}, err
	}
	if err := u.scan.Execute(ctx, v, []string{in.Path}); err != nil {
		return domain.FileStat{}, err
	}
	return u.stat.Execute(ctx, v, in.Path)
}

func verifyStagedHash(staged io.ReadSeeker, expectedHash string) error {
	hash, err := domain.HashReader(staged)
	if err != nil {
		return err
	}
	if hash != expectedHash {
		return plugin.ErrHashMismatch
	}
	_, err = staged.Seek(0, io.SeekStart)
	return err
}

func (u *PlaceStagedFile) checkTarget(ctx context.Context, v domain.Vault, path string, baseline *domain.FileStat) error {
	now, err := u.stat.Execute(ctx, v, path)
	switch {
	case baseline == nil && err == nil:
		return fmt.Errorf("%s: %w", path, vaultdomain.ErrAlreadyExists)
	case baseline == nil && errors.Is(err, edit.ErrNotExist):
		return nil
	case err != nil && errors.Is(err, edit.ErrNotExist):
		return fmt.Errorf("%s is gone since it was read: %w", path, edit.ErrStale)
	case err != nil:
		return err
	case baseline != nil && now.Hash != baseline.Hash:
		return fmt.Errorf("%s changed since it was read: %w", path, edit.ErrStale)
	}
	return nil
}
