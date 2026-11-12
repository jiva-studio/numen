// Package source holds the values of the sources a reader imports and searches, and their sentinel errors.
package source

import "errors"

// ErrNotADocument is bytes that are not a document the reader reads.
var ErrNotADocument = errors.New("not a document this reads")

// ErrOtherRecipe is a coarse pass that failed because the index holds its vectors under another
// recipe than the one asked for.
var ErrOtherRecipe = errors.New("the vectors are kept under another recipe")
