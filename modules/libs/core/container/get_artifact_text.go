package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

func (c Config) GetArtifactText(queries port.SourceQueries) *source.GetArtifactText {
	return source.NewGetArtifactText(queries, c.GetDerivedStores())
}
