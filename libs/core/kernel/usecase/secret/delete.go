package secret

import (
	"context"
	"errors"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// Delete removes a secret with its value and its permissions.
type Delete struct{ Operation }

// NewDelete makes the Delete use case over the operation.
func NewDelete(op Operation) *Delete { return &Delete{Operation: op} }

// Execute removes the entry first and then the value, and configures the plugins that could read it.
// When the value cannot be removed the entry returns to its place with its permissions, so the
// value stays named and the call can be repeated.
func (u *Delete) Execute(ctx context.Context, id string) error {
	return u.apply(ctx, func() ([]string, error) {
		var removed plugin.Secret
		var index int
		err := u.ledger.ChangeSecrets(func(list []plugin.Secret) ([]plugin.Secret, error) {
			index = getIndex(list, id)
			if index < 0 {
				return nil, newNotFoundError(id)
			}
			removed = list[index]
			return append(list[:index:index], list[index+1:]...), nil
		})
		if err != nil {
			return nil, err
		}
		if err := u.deleteValue(ctx, id); err != nil {
			return nil, errors.Join(err, u.restoreEntry(removed, index))
		}
		return nil, nil
	})
}

func (u *Delete) restoreEntry(entry plugin.Secret, index int) error {
	return u.ledger.ChangeSecrets(func(list []plugin.Secret) ([]plugin.Secret, error) {
		if getIndex(list, entry.ID) >= 0 {
			return list, nil
		}
		return slices.Insert(list, min(index, len(list)), entry), nil
	})
}
