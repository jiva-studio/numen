package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"

	"github.com/jiva-studio/numen/modules/libs/core/port/recognitionport"
)

// GetProblems holds the vault to the standard checks.
func (Config) GetProblems(problems recognitionport.ProblemQueries) *vault.GetProblems {
	return vault.NewStandardGetProblems(problems)
}
