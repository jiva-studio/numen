package cards

import (
	"context"
	"errors"
	"fmt"
	"path"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// ErrRowGone is a row the vault holds no file or folder at.
var ErrRowGone = errors.New("no such file or folder")

// FileStatter stats a path of the vault without opening it.
type FileStatter interface {
	StatFile(ctx context.Context, path string) (domain.FileStat, error)
}

// GetRowFolder is the folder a note made from a row of the file list goes in: the row itself when it is
// a folder, else the folder holding the file. A row at the top of the vault gives the empty path.
func GetRowFolder(ctx context.Context, files FileStatter, row string) (string, error) {
	if row == "" {
		return "", nil
	}
	_, err := files.StatFile(ctx, row)
	switch {
	case errors.Is(err, host.ErrNotExist):
		return "", fmt.Errorf("%s: %w", row, ErrRowGone)
	case errors.Is(err, host.ErrNotANote):
		return row, nil
	case err != nil:
		return "", err
	}
	if folder := path.Dir(row); folder != "." {
		return folder, nil
	}
	return "", nil
}
