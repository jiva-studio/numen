package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// Book reads a book that reflows from the file the vault holds it in. The
// reader of the format is a plugin, and a call made while none answers is refused
// with a *UnavailableError.
type Book interface {
	// GetBook is the shape of the book at a path. A path that holds no book is
	// refused with ErrNoBook, bytes that are no book with ErrNotADocument, and a
	// locked book with ErrEncrypted.
	GetBook(ctx context.Context, path string) (domain.BookShape, error)

	// ReadBookMarkup is the markup of one section of the book at a path, while the
	// file is the bytes the baseline names. A nil baseline names none. A section the
	// book lacks is refused with ErrNoBookEntry, and a file that is other bytes
	// with ErrBookChanged.
	ReadBookMarkup(ctx context.Context, path, section string, baseline *domain.Fingerprint) (string, error)

	// ReadEntry is the bytes of one entry of the archive of the book the baseline
	// names, while the file is those bytes. An entry the archive lacks is refused
	// with ErrNoBookEntry, and a file that is other bytes with ErrBookChanged.
	ReadEntry(ctx context.Context, baseline domain.Fingerprint, entry string) ([]byte, error)
}
