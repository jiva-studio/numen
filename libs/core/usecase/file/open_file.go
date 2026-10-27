package file

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// ErrChanged is a file that is no longer the one an address was given out for.
var ErrChanged = errors.New("the file changed since this address was given out")

// OpenFile opens one file of the vault to read a part of.
type OpenFile struct {
	readers vaultport.VaultReaders
}

func NewOpenFile(readers vaultport.VaultReaders) *OpenFile {
	return &OpenFile{readers: readers}
}

// OpenedFile is a file open to read, and what the vault says about it. Whoever opens it closes it.
type OpenedFile struct {
	Body    io.ReadSeekCloser
	Path    string
	ModTime time.Time
}

// Execute opens the file while its size and modification time are the ones named, and refuses it
// with ErrChanged otherwise. A zero modification time names a file nothing was said about.
func (u *OpenFile) Execute(ctx context.Context, v domain.Vault, path string, named domain.FileStat) (OpenedFile, error) {
	reader, err := u.readers.Open(v)
	if err != nil {
		return OpenedFile{}, err
	}
	found, err := reader.StatFile(ctx, path)
	if err != nil {
		return OpenedFile{}, err
	}
	if named.Size != found.Size || getStamp(named.ModTime) != getStamp(found.ModTime) {
		return OpenedFile{}, ErrChanged
	}
	// The file is opened by the path the vault reports for it.
	body, err := reader.Open(ctx, found.Path)
	if err != nil {
		return OpenedFile{}, err
	}
	return OpenedFile{Body: body, Path: found.Path, ModTime: found.ModTime}, nil
}

// getStamp is a modification time in nanoseconds since the epoch, and zero for none.
func getStamp(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixNano()
}
