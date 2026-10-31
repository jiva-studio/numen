package service

// cfiSelector locates a segment in a book: CFI is an EPUB Canonical Fragment Identifier.
type cfiSelector struct {
	CFI string `json:"cfi,omitempty"`
}
