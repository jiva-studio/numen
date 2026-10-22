package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"

	"github.com/jiva-studio/numen/modules/libs/core/port/sourceport"
)

// WriteCorrections tells changed, when it is not nil, the path of each file whose transcript was corrected.
func (c Config) WriteCorrections(queries sourceport.SourceQueries, changed func(path string)) *source.WriteCorrections {
	return source.NewWriteCorrections(queries, c.GetDerivedStores(), changed)
}
