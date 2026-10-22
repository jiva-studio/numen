package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"

	"github.com/jiva-studio/numen/modules/libs/core/port/sourceport"
)

func (c Config) GetDocumentList(queries sourceport.SourceQueries) *source.GetDocumentList {
	return source.NewGetDocumentList(queries)
}
