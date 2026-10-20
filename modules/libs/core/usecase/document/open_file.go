package document

import (
	"context"
	"io"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// OpenFile opens one file of the vault to read a part of.
type OpenFile struct {
	readers port.VaultReaders
}

func NewOpenFile(readers port.VaultReaders) *OpenFile {
	return &OpenFile{readers: readers}
}

// OpenedFile is a file open to read, and what the vault says about it. Whoever opens it closes it.
type OpenedFile struct {
	Body    io.ReadSeekCloser
	Path    string
	ModTime time.Time
}

// Execute opens the file while it is the bytes the named fingerprint gives, and refuses it with ErrChanged otherwise.
func (u *OpenFile) Execute(ctx context.Context, v domain.Vault, path string, named Fingerprint) (OpenedFile, error) {
	reader, mark, err := getFingerprint(ctx, u.readers, v, path)
	if err != nil {
		return OpenedFile{}, err
	}
	if err := checkUnchanged(named, mark); err != nil {
		return OpenedFile{}, err
	}
	// The file is opened by the path the vault reports for it.
	body, err := reader.Open(ctx, mark.Path)
	if err != nil {
		return OpenedFile{}, err
	}
	return OpenedFile{Body: body, Path: mark.Path, ModTime: modTime(mark.Mtime)}, nil
}

func modTime(nanos int64) time.Time {
	if nanos == 0 {
		return time.Time{}
	}
	return time.Unix(0, nanos)
}
