package secret

import (
	"context"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// Allow lets one declaration of a plugin read a secret.
type Allow struct{ Operation }

// NewAllow makes the Allow use case over the operation.
func NewAllow(op Operation) *Allow { return &Allow{Operation: op} }

// Execute adds the permission; one that is there already is kept as it is. It fails where the
// plugin is not installed or does not declare the name and group.
func (u *Allow) Execute(ctx context.Context, secretID string, permission plugin.DeclarationKey) error {
	return u.apply(ctx, func() ([]string, error) {
		if !slices.Contains(u.plugins.ListDeclarations(), permission) {
			return nil, newInvalidError("the plugin is not installed or does not declare that secret")
		}
		return nil, u.ledger.ChangeSecrets(func(list []plugin.Secret) ([]plugin.Secret, error) {
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
type Deny struct{ Operation }

// NewDeny makes the Deny use case over the operation.
func NewDeny(op Operation) *Deny { return &Deny{Operation: op} }

// Execute removes the permission; one that is not there is not an error.
func (u *Deny) Execute(ctx context.Context, secretID string, permission plugin.DeclarationKey) error {
	return u.apply(ctx, func() ([]string, error) {
		return nil, u.ledger.ChangeSecrets(func(list []plugin.Secret) ([]plugin.Secret, error) {
			index := getIndex(list, secretID)
			if index < 0 {
				return nil, newNotFoundError(secretID)
			}
			list[index].Allowed = slices.DeleteFunc(slices.Clone(list[index].Allowed), func(one plugin.DeclarationKey) bool {
				return one == permission
			})
			return list, nil
		})
	})
}
