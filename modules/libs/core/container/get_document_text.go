package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

func (c Config) GetDocumentText(queries port.SourceQueries) *source.GetDocumentText {
	return source.NewGetDocumentText(c.VaultReaders(), queries, c.GetDerivedStores(), c.TextExtractor())
}
