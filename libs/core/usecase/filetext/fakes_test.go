package filetext

import (
	"github.com/jiva-studio/numen/libs/core/internal/testsupport/indexfake"
	"github.com/jiva-studio/numen/libs/core/port/sourceport"
)

func newIndex(shelves indexfake.Vaults, sources sourceport.SourceRepository, queries sourceport.SourceQueries) Index {
	extract, _ := newIndexWithTexts(shelves, sources, queries)
	return extract
}

func newIndexWithTexts(shelves indexfake.Vaults, sources sourceport.SourceRepository, queries sourceport.SourceQueries) (Index, *libraryTexts) {
	texts := newLibraryTexts(shelves)
	return NewIndex(shelves, sources, queries, texts), texts
}
