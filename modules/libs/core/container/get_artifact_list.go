package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

func (c Config) GetArtifactList(queries port.SourceQueries) *source.GetArtifactList {
	return source.NewGetArtifactList(queries, c.GetDerivedStores())
}
