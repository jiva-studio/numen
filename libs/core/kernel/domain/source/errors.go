// Package source holds the sentinel errors of the documents a reader imports.
package source

import "errors"

// ErrNotADocument is bytes that are not a document the reader reads.
var ErrNotADocument = errors.New("not a document this reads")
