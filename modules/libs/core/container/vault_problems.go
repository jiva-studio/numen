package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

func (c Config) GetSummary(queries port.NoteQueries) *vault.GetSummary {
	return vault.NewGetSummary(queries)
}

func (c Config) ListProblems(queries port.NoteQueries) *vault.ListProblems {
	return vault.NewStandardListProblems(queries)
}
