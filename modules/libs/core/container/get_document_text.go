package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"

	"github.com/jiva-studio/numen/modules/libs/core/port/sourceport"
)

func (c Config) GetDocumentText(queries sourceport.SourceQueries) *source.GetDocumentText {
	return source.NewGetDocumentText(c.VaultReaders(), queries, c.GetDerivedStores(), c.TextExtractor())
}
