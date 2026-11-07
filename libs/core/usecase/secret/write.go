package secret

import "context"

// Write replaces the value of a secret.
type Write struct{ operation }

// Execute stores the new value and configures the plugins that may read the secret.
func (u *Write) Execute(ctx context.Context, id, value string) error {
	if value == "" {
		return invalid("the value is empty")
	}
	return u.plugins.ApplySecretChange(ctx, func() ([]string, error) {
		list, err := u.ledger.GetSecrets()
		if err != nil {
			return nil, err
		}
		index := getIndex(list, id)
		if index < 0 {
			return nil, notFound(id)
		}
		if err := u.setValue(ctx, id, value); err != nil {
			return nil, err
		}
		return listPluginIDs(list[index].Allowed), nil
	})
}
