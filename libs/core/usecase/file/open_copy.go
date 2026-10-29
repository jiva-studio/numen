package file

import (
	"context"
	"errors"
	"io"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/playbackport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// OpenProxy opens the copy a plugin keeps of a file of the vault, to be played in place of the file.
type OpenProxy struct {
	readers vaultport.VaultReaders
	copies  playbackport.Copies
}

func NewOpenProxy(readers vaultport.VaultReaders, copies playbackport.Copies) *OpenProxy {
	return &OpenProxy{readers: readers, copies: copies}
}

// CopyFile is a copy opened for partial reads, and the media type it is played
// as. The caller closes it.
type CopyFile struct {
	Body      io.ReadSeekCloser
	Size      int64
	MediaType string
}

// Execute opens the stored copy of the file at the path as it currently reads,
// and returns false when no copy is stored. A note is matched by the type in its
// frontmatter, so a plugin that stores copies of notes of its own type stores
// none of an ordinary note.
func (u *OpenProxy) Execute(ctx context.Context, v domain.Vault, path string) (CopyFile, bool, error) {
	var raw []byte
	var noteType domain.NoteType
	if IsNote(path) && u.copies.IsCopyingNotes() {
		read, err := u.read(ctx, v, path)
		if err != nil {
			return CopyFile{}, false, err
		}
		raw, noteType = read, getNoteType(read)
	}
	if !u.copies.IsCopied(path, noteType) {
		return CopyFile{}, false, nil
	}
	if raw == nil {
		read, err := u.read(ctx, v, path)
		if err != nil {
			return CopyFile{}, false, err
		}
		raw = read
	}
	found, err := u.copies.OpenProxy(ctx, v, path, noteType, domain.HashContent(raw))
	if errors.Is(err, playbackport.ErrNoProxy) {
		return CopyFile{}, false, nil
	}
	if err != nil {
		return CopyFile{}, false, err
	}
	return CopyFile{Body: found.Body, Size: found.Size, MediaType: found.MediaType}, true, nil
}

func (u *OpenProxy) read(ctx context.Context, v domain.Vault, path string) ([]byte, error) {
	reader, err := u.readers.Open(v)
	if err != nil {
		return nil, err
	}
	return reader.Read(ctx, path)
}
