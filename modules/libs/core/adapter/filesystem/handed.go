package filesystem

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Handed is what a person handed this application, on a machine where what
// they handed over is a path.
type Handed struct{}

func (Handed) Named(handle string) string { return filepath.Base(handle) }

func (Handed) Stat(_ context.Context, handle string) (port.HandedFile, error) {
	// A link is not followed: what it points at is not what was handed over.
	info, err := os.Lstat(handle)
	if err != nil {
		return port.HandedFile{}, err
	}
	return port.HandedFile{
		Name:   filepath.Base(handle),
		Handle: handle,
		Folder: info.IsDir(),
		File:   info.Mode().IsRegular(),
	}, nil
}

func (Handed) List(_ context.Context, handle string) ([]port.HandedFile, error) {
	held, err := os.ReadDir(handle)
	if err != nil {
		return nil, err
	}
	out := make([]port.HandedFile, 0, len(held))
	for _, one := range held {
		out = append(out, port.HandedFile{
			Name:   one.Name(),
			Handle: filepath.Join(handle, one.Name()),
		})
	}
	return out, nil
}

func (Handed) Open(_ context.Context, handle string) (io.ReadCloser, error) {
	return os.Open(handle)
}

func (Handed) Around(handle, vault string) bool {
	from, err := filepath.Abs(handle)
	if err != nil {
		return false
	}
	under, err := filepath.Abs(vault)
	if err != nil {
		return false
	}
	return under == from || strings.HasPrefix(under, from+string(filepath.Separator))
}
