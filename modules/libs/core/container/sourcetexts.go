package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// SourceTexts reads and writes the text made from a source, against the index that says which model produced it.
func (c Config) SourceTexts(sources port.SourceQueries) source.Texts {
	return source.NewTexts(c.VaultReaders(), c.VaultWriters(), sources, c.GetDerivedStores())
}
