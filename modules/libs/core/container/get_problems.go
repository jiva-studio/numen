package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/check"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// GetProblems holds the vault to the standard checks.
func (Config) GetProblems(problems port.ProblemQueries) *vault.GetProblems {
	return vault.NewGetProblems(check.Standard(problems))
}
