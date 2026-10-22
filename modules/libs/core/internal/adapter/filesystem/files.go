package filesystem

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/port/sourceport"
)

// ImportedFiles is what a person handed this application, on a machine where
// what they handed over is a path.
type ImportedFiles struct{}

func (ImportedFiles) GetName(handle string) string { return filepath.Base(handle) }

func (ImportedFiles) GetImportedFile(_ context.Context, handle string) (sourceport.ImportedFile, error) {
	// A link is not followed: what it points at is not what was handed over.
	info, err := os.Lstat(handle)
	if err != nil {
		return sourceport.ImportedFile{}, err
	}
	return sourceport.ImportedFile{
		Name:     filepath.Base(handle),
		Handle:   handle,
		IsFolder: info.IsDir(),
		IsFile:   info.Mode().IsRegular(),
	}, nil
}

func (ImportedFiles) List(_ context.Context, handle string) ([]sourceport.ImportedFile, error) {
	target, err := os.ReadDir(handle)
	if err != nil {
		return nil, err
	}
	out := make([]sourceport.ImportedFile, 0, len(target))
	for _, one := range target {
		out = append(out, sourceport.ImportedFile{
			Name:   one.Name(),
			Handle: filepath.Join(handle, one.Name()),
		})
	}
	return out, nil
}

func (ImportedFiles) Open(_ context.Context, handle string) (io.ReadCloser, error) {
	return os.Open(handle)
}

func (ImportedFiles) HasHandle(handle, vault string) bool {
	from, err := filepath.Abs(handle)
	if err != nil {
		return false
	}
	limit, err := filepath.Abs(vault)
	if err != nil {
		return false
	}
	return limit == from || strings.HasPrefix(limit, from+string(filepath.Separator))
}
