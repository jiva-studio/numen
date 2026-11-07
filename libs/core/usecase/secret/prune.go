package secret

import "slices"

// Prune removes the permissions of declarations that are gone.
type Prune struct{ operation }

// Execute removes every permission whose declaration is not among those given: a plugin that was
// uninstalled, or a declaration an update dropped.
func (u *Prune) Execute(declared []Permission) error {
	return u.ledger.ChangeSecrets(func(list []Secret) ([]Secret, error) {
		pruned := slices.Clone(list)
		for index := range pruned {
			pruned[index].Allowed = slices.DeleteFunc(slices.Clone(pruned[index].Allowed), func(one Permission) bool {
				return !slices.Contains(declared, one)
			})
		}
		return pruned, nil
	})
}
