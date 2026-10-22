package document

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"

	"github.com/jiva-studio/numen/modules/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/modules/libs/core/port/vaultport"
)

// ErrNotAPicture is an entry of an archive that is not one of the pictures a
// book is drawn from.
var ErrNotAPicture = errors.New("this entry is not a picture")

// Picture is an entry of a book's archive, and the type its bytes settle.
type Picture struct {
	Type string
	Body []byte
}

// pictured are the types an entry may be served as. SVG is a document, with
// script, style and addresses of its own, and is not among them.
var pictured = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/gif":  true,
	"image/webp": true,
}

// GetBookPicture reads one picture out of a book's archive.
type GetBookPicture struct {
	readers vaultport.VaultReaders
	books   sourceport.Book
	detect  func([]byte) string
}

// NewGetBookPicture settles the type of an entry by detect, which reads the bytes.
func NewGetBookPicture(readers vaultport.VaultReaders, books sourceport.Book, detect func([]byte) string) GetBookPicture {
	return GetBookPicture{readers: readers, books: books, detect: detect}
}

// Execute answers while the file at the path is the bytes the named fingerprint
// gives, and refuses it with ErrChanged otherwise. The bytes are somebody
// else's, so the type is settled by reading them, and an entry that is none of
func (u GetBookPicture) Execute(
	ctx context.Context, v domain.Vault, path string, named Fingerprint, entry string,
) (Picture, error) {
	_, mark, err := getFingerprint(ctx, u.readers, v, path)
	if err != nil {
		return Picture{}, err
	}
	if err := checkUnchanged(named, mark); err != nil {
		return Picture{}, err
	}
	body, err := u.books.ReadEntry(ctx, getDomainFingerprint(mark), entry)
	if errors.Is(err, sourceport.ErrBookChanged) {
		return Picture{}, ErrChanged
	}
	if err != nil {
		return Picture{}, err
	}
	kind := u.detect(body)
	if !pictured[kind] {
		return Picture{}, ErrNotAPicture
	}
	return Picture{Type: kind, Body: body}, nil
}
