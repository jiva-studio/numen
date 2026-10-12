package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// WriteCorrections tells changed, when it is not nil, the path of each file whose transcript was corrected.
func (c Config) WriteCorrections(queries port.SourceQueries, changed func(path string)) *source.WriteCorrections {
	return source.NewWriteCorrections(queries, c.GetDerivedStores(), changed)
}
