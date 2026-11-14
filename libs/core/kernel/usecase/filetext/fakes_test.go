package filetext

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/chunking"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/indexfake"
)

func newIndex(shelves indexfake.Vaults, index *indexfake.Store) Index {
	extract, _ := newIndexWithTexts(shelves, index)
	return extract
}

func newIndexWithTexts(shelves indexfake.Vaults, index *indexfake.Store) (Index, *libraryTexts) {
	texts := newLibraryTexts(shelves)
	return NewIndex(shelves, index, index, index, index, texts, chunking.Sizes{}, chunking.Legibility{}, false), texts
}
