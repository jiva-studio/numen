// Package opf reads the package document of an EPUB: metadata, manifest, spine and layout.
package opf

// A Layout is how a book's pages are drawn: as reflowable text or as fixed pages.
type Layout string

const (
	// Reflowable is a layout whose text flows to fit the window.
	Reflowable Layout = "reflowable"
	// PrePaginated is a layout of fixed pages.
	PrePaginated Layout = "pre-paginated"
)

// A Direction is the direction a book's pages progress in.
type Direction string

const (
	// DefaultDirection is the direction of a book that specifies none; a reader lays it out in the
	// direction of its language.
	DefaultDirection Direction = ""
	// LeftToRight is the direction "ltr".
	LeftToRight Direction = "ltr"
	// RightToLeft is the direction "rtl".
	RightToLeft Direction = "rtl"
)
