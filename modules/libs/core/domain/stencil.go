package domain

// Stencil is one stencil as a caller choosing between them sees it: where the
// file is, and what it is called.
type Stencil struct {
	Path  string
	Title string
}
