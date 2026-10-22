package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"

	"github.com/jiva-studio/numen/modules/libs/core/port/noteport"
)

func (Config) GetSummary(queries noteport.NoteQueries) *vault.GetSummary {
	return vault.NewGetSummary(queries)
}
