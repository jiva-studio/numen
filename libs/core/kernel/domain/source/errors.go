// Package source holds the values of the sources a reader imports and searches, and their sentinel errors.
package source

import "errors"

// ErrNotADocument is bytes that are not a document the reader reads.
var ErrNotADocument = errors.New("not a document this reads")
