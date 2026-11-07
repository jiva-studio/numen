package secret

import "context"

// Rename changes the name of a secret. Its id, value and permissions stay.
type Rename struct{ operation }

// Execute renames the secret in the list; the key store is not touched.
func (u *Rename) Execute(ctx context.Context, id, name string) error {
	if err := checkName("name", name); err != nil {
		return err
	}
	return u.plugins.ApplySecretChange(ctx, func() ([]string, error) {
		return nil, u.ledger.ChangeSecrets(func(list []Secret) ([]Secret, error) {
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
