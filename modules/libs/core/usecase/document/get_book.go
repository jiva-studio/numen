package document

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// GetBook reads what a book that reflows is.
type GetBook struct {
	books port.Book
}

// NewGetBook reads books through books.
func NewGetBook(books port.Book) GetBook {
	return GetBook{books: books}
}

// Execute answers the shape of the book at a path. It refuses a path that holds
// no book with port.ErrNoBook, and a book it cannot read with port.ErrNotADocument
// or port.ErrEncrypted.
func (u GetBook) Execute(ctx context.Context, path string) (domain.BookShape, error) {
	return u.books.GetBook(ctx, path)
}

// ReadBookMarkup reads one section of a book that reflows as markup.
type ReadBookMarkup struct {
	books port.Book
}

// NewReadBookMarkup reads books through books.
func NewReadBookMarkup(books port.Book) ReadBookMarkup {
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
