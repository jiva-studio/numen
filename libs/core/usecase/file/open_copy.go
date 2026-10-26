package file

import (
	"context"
	"errors"
	"io"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/playbackport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// OpenCopy opens the copy a plugin keeps of a file of the vault, to be played in place of the file.
type OpenCopy struct {
	readers vaultport.VaultReaders
	copies  playbackport.Copies
}

func NewOpenCopy(readers vaultport.VaultReaders, copies playbackport.Copies) *OpenCopy {
	return &OpenCopy{readers: readers, copies: copies}
}

// CopyFile is a copy open to read a part of, and the media type it is played as. Whoever opens it
// closes it.
type CopyFile struct {
	Body      io.ReadSeekCloser
	Size      int64
	MediaType string
}

// Execute opens the copy kept of the file at the path as it now reads, and answers false where no
// copy of it is kept.
func (u *OpenCopy) Execute(ctx context.Context, v domain.Vault, path string) (CopyFile, bool, error) {
	if !u.copies.IsCopied(path) {
		return CopyFile{}, false, nil
	}
	reader, err := u.readers.Open(v)
	if err != nil {
		return CopyFile{}, false, err
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return CopyFile{}, false, err
	}
	found, err := u.copies.OpenCopy(ctx, v, path, domain.HashContent(raw))
	if errors.Is(err, playbackport.ErrNoCopy) {
		return CopyFile{}, false, nil
	}
	if err != nil {
		return CopyFile{}, false, err
	}
	return CopyFile{Body: found.Body, Size: found.Size, MediaType: found.MediaType}, true, nil
}
