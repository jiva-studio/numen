package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// Overview builds what says how much a vault holds and what in it could not be
// acted on, over the index.
func (c Config) Overview(queries port.NoteQueries) vault.Overview {
	return vault.NewOverview(queries)
}
