package document

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/sourceport"
)

type GetBook struct {
	books sourceport.Book
}

// NewGetBook reads books through books.
func NewGetBook(books sourceport.Book) GetBook {
	return GetBook{books: books}
}

// Execute answers the shape of the book at a path. It refuses a path that holds
// no book with port.ErrNoBook, and a book it cannot read with port.ErrNotADocument
// or port.ErrEncrypted.
func (u GetBook) Execute(ctx context.Context, path string) (domain.BookShape, error) {
	return u.books.GetBook(ctx, path)
}

type ReadBookMarkup struct {
	books sourceport.Book
}

// NewReadBookMarkup reads books through books.
func NewReadBookMarkup(books sourceport.Book) ReadBookMarkup {
	return ReadBookMarkup{books: books}
}

// Execute answers the markup of one section of the book at a path, while the file
// is the bytes the baseline names. It refuses a section the book lacks with
// ErrNoEntry, and a file that is other bytes with port.ErrBookChanged.
func (u ReadBookMarkup) Execute(
	ctx context.Context, path, section string, baseline *domain.Fingerprint,
) (string, error) {
	return u.books.ReadBookMarkup(ctx, path, section, baseline)
}
