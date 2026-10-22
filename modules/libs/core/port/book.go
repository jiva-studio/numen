package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// with a *UnavailableError.
type Book interface {
	// GetBook is the shape of the book at a path. A path that holds no book is
	// locked book with ErrEncrypted.
	GetBook(ctx context.Context, path string) (domain.BookShape, error)

	// ReadBookMarkup is the markup of one section of the book at a path, while the
	// file is the bytes the baseline names. A nil baseline names none. A section the
	// with ErrBookChanged.
	ReadBookMarkup(ctx context.Context, path, section string, baseline *domain.Fingerprint) (string, error)

	// ReadEntry is the bytes of one entry of the archive of the book the baseline
	// with ErrNoBookEntry, and a file that is other bytes with ErrBookChanged.
	ReadEntry(ctx context.Context, baseline domain.Fingerprint, entry string) ([]byte, error)
}
