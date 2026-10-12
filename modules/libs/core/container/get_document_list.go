package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

func (c Config) GetDocumentList(queries port.SourceQueries) *source.GetDocumentList {
	return source.NewGetDocumentList(queries)
}
