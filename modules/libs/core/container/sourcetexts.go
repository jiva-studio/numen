package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"

	"github.com/jiva-studio/numen/modules/libs/core/port/sourceport"
)

// SourceTexts reads and writes the text made from a source, against the index that says which model produced it.
func (c Config) SourceTexts(sources sourceport.SourceQueries) source.Texts {
	return source.NewTexts(c.VaultReaders(), c.VaultWriters(), sources, c.GetDerivedStores())
}
