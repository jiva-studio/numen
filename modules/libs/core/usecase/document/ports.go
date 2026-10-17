package document

import (
	"context"
	"image"
	"io"
)

// Scan is a document held open, for the length of one call.
type Scan interface {
	GetPageCount() int
	GetPageSize(index int) (wide, high float64, err error)
	// Render draws one page as wide as was asked for, in pixels.
	Render(index, width int) (image.Image, error)
}

// Documents holds documents open.
type Documents interface {
	// UseDocument calls use with the document at a fingerprint, opening it from
	// the file where it is not open, and answers while nobody else is using it.
	UseDocument(
		ctx context.Context, mark Fingerprint,
		file func(context.Context) (io.ReadSeekCloser, error),
		use func(Scan) error,
	) error
}

// Books reads the archives of books.
type Books interface {
	// ReadEntry answers the bytes of one entry of the archive of the book at a
	// fingerprint. It refuses an entry the archive lacks with ErrNoEntry, and a
	// file that is other bytes now with port.ErrBookChanged.
	ReadEntry(ctx context.Context, mark Fingerprint, entry string) ([]byte, error)
}

// PageKey is one drawn page: which document, which page, and how wide it was
// drawn.
type PageKey struct {
	Document Fingerprint
	Page     int
	Width    int
}

// Pages holds the pages already drawn.
type Pages interface {
	// GetPage answers the page at a key, and draws it where it is not held.
	GetPage(ctx context.Context, key PageKey, draw func() ([]byte, error)) ([]byte, error)
}
