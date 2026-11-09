package secret

import (
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

// Prune removes the permissions of declarations that are gone.
type Prune struct{ operation }

// Execute removes every permission whose declaration is not among those given: a plugin that was
// uninstalled, or a declaration an update dropped.
func (u *Prune) Execute(declared []pluginport.DeclarationKey) error {
	return u.ledger.ChangeSecrets(func(list []pluginport.Secret) ([]pluginport.Secret, error) {
		pruned := slices.Clone(list)
		for index := range pruned {
			pruned[index].Allowed = slices.DeleteFunc(slices.Clone(pruned[index].Allowed), func(one pluginport.DeclarationKey) bool {
				return !slices.Contains(declared, one)
			})
		}
		return pruned, nil
	})
}
