package document

import (
	"bytes"
	"context"
	"errors"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// ErrNotAPicture is an entry of an archive that is not one of the pictures a
// book is drawn from.
var ErrNotAPicture = errors.New("this entry is not a picture")

// Picture is an entry of a book's archive, and the type its bytes settle.
type Picture struct {
	Type string
	Body []byte
}

// GetBookPicture reads one picture out of a book's archive.
type GetBookPicture struct {
	readers port.VaultReaders
	books   Books
}

func NewGetBookPicture(readers port.VaultReaders, books Books) GetBookPicture {
	return GetBookPicture{readers: readers, books: books}
}

// Execute answers while the file at the path is the bytes the named fingerprint
// gives, and refuses it with ErrChanged otherwise. The bytes are somebody
// else's, so the type is settled by reading them, and an entry that is none of
// the pictures is refused with ErrNotAPicture.
func (u GetBookPicture) Execute(
	ctx context.Context, v domain.Vault, path string, named Fingerprint, entry string,
) (Picture, error) {
	read, _, err := readBook(ctx, u.readers, u.books, v, path, &named)
	if err != nil {
		return Picture{}, err
	}
	body, err := read.Entry(entry)
	if err != nil {
		return Picture{}, err
	}
	drawn := getPictureType(body)
	if drawn == "" {
		return Picture{}, ErrNotAPicture
	}
	return Picture{Type: drawn, Body: body}, nil
}

// getPictureType is the type the leading bytes of a picture settle, and nothing
// for any other. SVG is a document, with script, style and addresses of its own,
// and is none of them.
func getPictureType(body []byte) string {
	switch {
	case bytes.HasPrefix(body, []byte("\x89PNG\r\n\x1a\n")):
		return "image/png"
	case bytes.HasPrefix(body, []byte("\xff\xd8\xff")):
		return "image/jpeg"
	case bytes.HasPrefix(body, []byte("GIF87a")), bytes.HasPrefix(body, []byte("GIF89a")):
		return "image/gif"
	case len(body) >= 12 && bytes.HasPrefix(body, []byte("RIFF")) && bytes.Equal(body[8:12], []byte("WEBP")):
		return "image/webp"
	default:
		return ""
	}
}
