package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

func (c Config) GetSummary(queries port.NoteQueries) *vault.GetSummary {
	return vault.NewGetSummary(queries)
}

func (c Config) GetProblems(queries port.NoteQueries) *vault.GetProblems {
	return vault.NewStandardGetProblems(queries)
}
