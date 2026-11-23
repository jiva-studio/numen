package hostvault

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
)

// MoveFile moves a file, or a folder with everything under it, as the window does: the notes that travel are filed
// and the links to them repaired. It returns the count of files under to.
func (w Writer) MoveFile(ctx context.Context, from, to string, baseline *domain.FileStat) (int, error) {
	v, err := w.session.open()
	if err != nil {
		return 0, err
	}
	count, err := w.Move.Execute(ctx, v, from, to, baseline)
	w.session.ReportWrites(v.ID, err)
	return count, err
}

// RemoveFile moves a file, or a folder with everything under it, to the trash.
func (w Writer) RemoveFile(ctx context.Context, path string, baseline *domain.FileStat) (int, error) {
	v, err := w.session.open()
	if err != nil {
		return 0, err
	}
	count, err := w.Remove.Execute(ctx, v, path, baseline)
	w.session.ReportWrites(v.ID, err)
	return count, err
}

// PlaceFile puts the staged file of the plugin in the vault at path and drops it from the store folder.
func (w Writer) PlaceFile(
	ctx context.Context, pluginID, name, path, expectedHash string, baseline *domain.FileStat,
) (domain.FileStat, error) {
	v, err := w.session.open()
	if err != nil {
		return domain.FileStat{}, err
	}
	in := file.PlaceInput{Path: path, ExpectedHash: expectedHash, Baseline: baseline}
	stat, err := w.place(ctx, v, pluginID, name, in)
	w.session.ReportWrites(v.ID, err)
	if err != nil {
		return domain.FileStat{}, err
	}
	writer, err := w.StoreWriters(v, pluginID)
	if err != nil {
		return domain.FileStat{}, err
	}
	if err := writer.Delete(ctx, name); err != nil {
		return domain.FileStat{}, err
	}
	return stat, nil
}

func (w Writer) place(
	ctx context.Context, v domain.Vault, pluginID, name string, in file.PlaceInput,
) (_ domain.FileStat, err error) {
	reader, err := w.StoreReaders(v, pluginID)
	if err != nil {
		return domain.FileStat{}, err
	}
	staged, _, err := reader.Open(ctx, name)
	if err != nil {
		return domain.FileStat{}, err
	}
	defer func() { err = errors.Join(err, staged.Close()) }()
	in.Staged = staged
	return w.Place.Execute(ctx, v, in)
}
