package secret

import "context"

// Delete removes a secret with its value and its permissions.
type Delete struct{ operation }

// Execute removes the entry first and then the value, and configures the plugins that could read it.
func (u *Delete) Execute(ctx context.Context, id string) error {
	return u.plugins.ApplySecretChange(ctx, func() ([]string, error) {
		err := u.ledger.ChangeSecrets(func(list []Secret) ([]Secret, error) {
			index := getIndex(list, id)
			if index < 0 {
				return nil, notFound(id)
			}
			return append(list[:index:index], list[index+1:]...), nil
		})
		if err != nil {
			return nil, err
		}
		return nil, u.deleteValue(ctx, id)
	})
}
