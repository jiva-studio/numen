package secret

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// Rename changes the name of a secret. Its id, value and permissions stay.
type Rename struct{ Operation }

// NewRename makes the Rename use case over the operation.
func NewRename(op Operation) *Rename { return &Rename{Operation: op} }

// Execute renames the secret in the list; the key store is not touched.
func (u *Rename) Execute(ctx context.Context, id, name string) error {
	if err := checkName("name", name); err != nil {
		return err
	}
	return u.apply(ctx, func() ([]string, error) {
		return nil, u.ledger.ChangeSecrets(func(list []plugin.Secret) ([]plugin.Secret, error) {
			index := getIndex(list, id)
			if index < 0 {
				return nil, newNotFoundError(id)
			}
			if isNameTaken(list, list[index].Group, name, id) {
				return nil, newInvalidError("the name is taken in the group")
			}
			list[index].Name = name
			return list, nil
		})
	})
}
