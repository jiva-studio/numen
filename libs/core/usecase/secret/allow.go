package secret

import (
	"context"
	"slices"
)

// Allow lets one declaration of a plugin read a secret.
type Allow struct{ operation }

// Execute adds the permission; one that is there already is kept as it is. It fails where the
// plugin is not installed or does not declare the name and group.
func (u *Allow) Execute(ctx context.Context, secretID string, permission Permission) error {
	return u.plugins.ApplySecretChange(ctx, func() ([]string, error) {
		if !slices.Contains(u.plugins.ListDeclarations(), permission) {
			return nil, newInvalidError("the plugin is not installed or does not declare that secret")
		}
		return nil, u.ledger.ChangeSecrets(func(list []Secret) ([]Secret, error) {
			index := getIndex(list, secretID)
			if index < 0 {
				return nil, newNotFoundError(secretID)
			}
			if !slices.Contains(list[index].Allowed, permission) {
				list[index].Allowed = append(slices.Clone(list[index].Allowed), permission)
			}
			return list, nil
		})
	})
}

// Deny stops one declaration of a plugin from reading a secret.
type Deny struct{ operation }

// Execute removes the permission; one that is not there is not an error.
func (u *Deny) Execute(ctx context.Context, secretID string, permission Permission) error {
	return u.plugins.ApplySecretChange(ctx, func() ([]string, error) {
		return nil, u.ledger.ChangeSecrets(func(list []Secret) ([]Secret, error) {
			index := getIndex(list, secretID)
			if index < 0 {
				return nil, newNotFoundError(secretID)
			}
			list[index].Allowed = slices.DeleteFunc(slices.Clone(list[index].Allowed), func(one Permission) bool {
				return one == permission
			})
			return list, nil
		})
	})
}
