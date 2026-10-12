package document

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/epub"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Book is a book read, with the fingerprint of the file it was read from.
type Book struct {
	*epub.Book
	Fingerprint Fingerprint
}

// GetBook reads the book at a path.
type GetBook struct {
	readers port.VaultReaders
	books   Books
}

func NewGetBook(readers port.VaultReaders, books Books) GetBook {
	return GetBook{readers: readers, books: books}
}

func (u GetBook) Execute(ctx context.Context, v domain.Vault, path string) (Book, error) {
	read, mark, err := u.read(ctx, v, path)
	if err != nil {
		return Book{}, err
	}
	return Book{Book: read, Fingerprint: mark}, nil
}

// read answers the book at a path, held where it is read already.
//
// The bytes are read where the book is not held, and the read outlives the
// request that asked for it: a caller that ran out of patience is told the book
// is busy, and the book is there for the next ask.
func (u GetBook) read(ctx context.Context, v domain.Vault, path string) (*epub.Book, Fingerprint, error) {
	return readBook(ctx, u.readers, u.books, v, path, nil)
}

// readBook is the book at a path, refused with ErrChanged where the file is not
// the bytes the baseline names.
func readBook(
	ctx context.Context, readers port.VaultReaders, books Books,
	v domain.Vault, path string, baseline *Fingerprint,
) (*epub.Book, Fingerprint, error) {
	reader, mark, err := getFingerprint(ctx, readers, v, path)
	if err != nil {
		return nil, Fingerprint{}, err
	}
	if baseline != nil {
		if err := checkUnchanged(*baseline, mark); err != nil {
			return nil, Fingerprint{}, err
		}
	}
	read, err := books.GetBook(ctx, mark, func() (*epub.Book, error) {
		raw, err := reader.Read(context.WithoutCancel(ctx), mark.Path)
		if err != nil {
			return nil, err
		}
		return epub.Read(raw)
	})
	if err != nil {
		return nil, Fingerprint{}, err
	}
	return read, mark, nil
}
