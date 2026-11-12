package secret

import "context"

// Write replaces the value of a secret.
type Write struct{ Operation }

// NewWrite makes the Write use case over the operation.
func NewWrite(op Operation) *Write { return &Write{Operation: op} }

// Execute stores the new value and configures the plugins that may read the secret.
func (u *Write) Execute(ctx context.Context, id, value string) error {
	if value == "" {
		return newInvalidError("the value is empty")
	}
	return u.apply(ctx, func() ([]string, error) {
		list, err := u.ledger.GetSecrets()
		if err != nil {
			return nil, err
		}
		index := getIndex(list, id)
		if index < 0 {
			return nil, newNotFoundError(id)
		}
		if err := u.setValue(ctx, id, value); err != nil {
			return nil, err
		}
		return listPluginIDs(list[index].Allowed), nil
	})
}
