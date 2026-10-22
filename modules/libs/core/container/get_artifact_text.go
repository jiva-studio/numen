package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"

	"github.com/jiva-studio/numen/modules/libs/core/port/sourceport"
)

func (c Config) GetArtifactText(queries sourceport.SourceQueries) *source.GetArtifactText {
	return source.NewGetArtifactText(queries, c.GetDerivedStores())
}
