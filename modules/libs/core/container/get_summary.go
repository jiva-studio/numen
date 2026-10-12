package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

func (Config) GetSummary(queries port.NoteQueries) *vault.GetSummary {
	return vault.NewGetSummary(queries)
}
