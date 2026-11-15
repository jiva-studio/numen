package filetext

import (
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/indexfake"
)

func newIndex(shelves indexfake.Vaults, index *indexfake.Store) Index {
	extract, _ := newIndexWithTexts(shelves, index)
	return extract
}

func newIndexWithTexts(shelves indexfake.Vaults, index *indexfake.Store) (Index, *libraryTexts) {
	texts := newLibraryTexts(shelves)
	return NewIndex(IndexDeps{Readers: shelves, Sources: index, Records: index, Backlog: index, Lookup: index, Texts: texts}), texts
}
