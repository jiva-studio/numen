package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"

	"github.com/jiva-studio/numen/modules/libs/core/port/sourceport"
)

func (c Config) GetArtifactList(queries sourceport.SourceQueries) *source.GetArtifactList {
	return source.NewGetArtifactList(queries, c.GetDerivedStores())
}
