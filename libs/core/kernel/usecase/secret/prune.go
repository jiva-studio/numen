package secret

import (
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// Prune removes the permissions of declarations that are gone.
type Prune struct{ Operation }

// NewPrune makes the Prune use case over the operation.
func NewPrune(op Operation) *Prune { return &Prune{Operation: op} }

// Execute removes every permission whose declaration is not among those given: a plugin that was
// uninstalled, or a declaration an update dropped.
func (u *Prune) Execute(declared []plugin.DeclarationKey) error {
	return u.ledger.ChangeSecrets(func(list []plugin.Secret) ([]plugin.Secret, error) {
		pruned := slices.Clone(list)
		for index := range pruned {
			pruned[index].Allowed = slices.DeleteFunc(slices.Clone(pruned[index].Allowed), func(one plugin.DeclarationKey) bool {
				return !slices.Contains(declared, one)
			})
		}
		return pruned, nil
	})
}
