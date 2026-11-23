package hostvault

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
)

// MoveFile moves a file, or a folder with everything under it, as the window does: the notes that travel are filed
// and the links to them repaired. It returns the count of files under to.
func (w Writer) MoveFile(ctx context.Context, from, to string, baseline *domain.FileStat) (int, error) {
	v, err := w.session.open()
	if err != nil {
		return 0, err
	}
	count, err := w.checkEntry(ctx, v, from, baseline)
	if err != nil {
		return 0, err
	}
	_, _, err = w.Move.Execute(ctx, v, from, to)
	if errors.Is(err, edit.ErrNoNote) {
		err = fmt.Errorf("%s: %w", from, fs.ErrNotExist)
	}
	w.session.ReportWrites(v.ID, err)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// RemoveFile moves a file, or a folder with everything under it, to the trash.
func (w Writer) RemoveFile(ctx context.Context, path string, baseline *domain.FileStat) (int, error) {
	v, err := w.session.open()
	if err != nil {
		return 0, err
	}
	count, err := w.checkEntry(ctx, v, path, baseline)
	if err != nil {
		return 0, err
	}
	_, _, err = w.Remove.Execute(ctx, v, path)
	w.session.ReportWrites(v.ID, err)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// checkEntry counts the files at the path and refuses a baseline the file no longer matches. A folder has no baseline.
func (w Writer) checkEntry(ctx context.Context, v domain.Vault, path string, baseline *domain.FileStat) (int, error) {
	count, err := w.Count.Execute(ctx, v, path)
	if err != nil {
		return 0, err
	}
	if count == 1 {
		return 1, w.checkBaseline(ctx, v, path, baseline)
	}
	return count, nil
}

func (w Writer) checkBaseline(ctx context.Context, v domain.Vault, path string, baseline *domain.FileStat) error {
	if baseline == nil {
		return nil
	}
	now, err := w.Stat.Execute(ctx, v, path)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%s is gone since it was read: %w", path, edit.ErrStale)
	}
	if err != nil {
		return err
	}
	if now.Hash != baseline.Hash {
		return fmt.Errorf("%s changed since it was read: %w", path, edit.ErrStale)
	}
	return nil
}

// PlaceFile puts the staged file of the plugin in the vault at path. The content goes to a temporary file next to
// the target and is renamed over it, so no reader sees a file that is not whole.
func (w Writer) PlaceFile(
	ctx context.Context, pluginID, name, path, expectedHash string, baseline *domain.FileStat,
) (domain.FileStat, error) {
	v, err := w.session.open()
	if err != nil {
		return domain.FileStat{}, err
	}
	if err := w.copyStaged(ctx, v, pluginID, name, path, expectedHash, baseline); err != nil {
		return domain.FileStat{}, err
	}
	err = w.Index.Execute(ctx, v, []string{path})
	w.session.ReportWrites(v.ID, err)
	if err != nil {
		return domain.FileStat{}, err
	}
	return w.finishPlacing(ctx, v, pluginID, name, path)
}

func (w Writer) copyStaged(
	ctx context.Context, v domain.Vault, pluginID, name, path, expectedHash string, baseline *domain.FileStat,
) (err error) {
	store, err := w.StoreReaders(v, pluginID)
	if err != nil {
		return err
	}
	staged, _, err := store.Open(ctx, name)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, staged.Close()) }()
	if err := checkStaged(staged, expectedHash); err != nil {
		return err
	}
	if err := w.checkTarget(ctx, v, path, baseline); err != nil {
		return err
	}
	return w.Import.Execute(ctx, v, path, staged, baseline != nil)
}

func checkStaged(staged io.ReadSeeker, expectedHash string) error {
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

func (w Writer) checkTarget(ctx context.Context, v domain.Vault, path string, baseline *domain.FileStat) error {
	if baseline != nil {
		return w.checkBaseline(ctx, v, path, baseline)
	}
	_, err := w.Stat.Execute(ctx, v, path)
	switch {
	case err == nil:
		return fmt.Errorf("%s: %w", path, vaultdomain.ErrAlreadyExists)
	case errors.Is(err, fs.ErrNotExist):
		return nil
	}
	return err
}

func (w Writer) finishPlacing(ctx context.Context, v domain.Vault, pluginID, name, path string) (domain.FileStat, error) {
	store, err := w.StoreWriters(v, pluginID)
	if err != nil {
		return domain.FileStat{}, err
	}
	if err := store.Delete(ctx, name); err != nil {
		return domain.FileStat{}, err
	}
	return w.Stat.Execute(ctx, v, path)
}

// IndexFollow says whether a write reached the vault and the index did not follow.
type IndexFollow struct{}

func (IndexFollow) IsIndexOutdated(err error) bool { return errors.Is(err, edit.ErrIndexOutdated) }
