package source

import "github.com/jiva-studio/numen/libs/notes/domain"

// Passage is where in a document to look. A Length of zero opens the document
// at its beginning, and Also is the other passages of the document to light.
type Passage struct {
	Path   string
	Start  int
	Length int
	Also   []domain.ByteSpan
}
