package filetext

import (
	"github.com/jiva-studio/numen/libs/core/internal/testsupport/indexfake"
	"github.com/jiva-studio/numen/libs/core/port/sourceport"
)

// newIndex is the cut of a library whose books have the text versions the
// stand-in reads.
func newIndex(shelves indexfake.Vaults, sources sourceport.SourceRepository, queries sourceport.SourceQueries) Index {
	extract, _ := newIndexWithTexts(shelves, sources, queries)
	return extract
}

// newIndexWithTexts is newIndex, and the text versions a test puts more of.
func newIndexWithTexts(shelves indexfake.Vaults, sources sourceport.SourceRepository, queries sourceport.SourceQueries) (Index, *libraryTexts) {
	texts := newLibraryTexts(shelves)
	return NewIndex(shelves, sources, queries, texts), texts
}
