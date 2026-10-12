package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// Artifacts builds everything that lists, reads and corrects what was made from
// a vault's files, against this installation's derived stores. Changed, when it
// is not nil, is told the path of each file whose transcript was corrected.
func (c Config) Artifacts(queries port.SourceQueries, changed func(path string)) source.Artifacts {
	derived := c.GetDerivedStores()
	return source.Artifacts{
		List:  source.NewListArtifacts(queries, derived),
		Read:  source.NewReadArtifact(queries, derived),
		Write: source.NewWriteCorrections(queries, derived, changed),
	}
}
