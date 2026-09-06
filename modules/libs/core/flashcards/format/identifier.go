package format

import "github.com/jiva-studio/numen/modules/libs/core/markdown"

// stamped writes an identifier into a file that carries none.
func stamped(doc *markdown.Document, identifier string) (bool, error) {
	if _, carried := doc.Identifier(); carried {
		return false, nil
	}
	return true, doc.SetIdentifier(identifier)
}
