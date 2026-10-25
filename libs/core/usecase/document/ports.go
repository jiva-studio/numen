package document

import (
	"context"
	"image"
	"io"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// VaultReaders reads the files of a vault.
type VaultReaders = vaultport.VaultReaders

// Scan represents an open document during a single operation.
type Scan interface {
	GetPageCount() int
	GetPageSize(index int) (width, height float64, err error)
	// Render draws one page scaled to target pixel width.
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
