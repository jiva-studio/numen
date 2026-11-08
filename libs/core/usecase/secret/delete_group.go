package secret

import (
	"context"
	"errors"
	"slices"
)

// DeleteGroup removes every secret of a group with its values and its permissions.
type DeleteGroup struct{ operation }

// Execute removes the entries of the group first and then the values, and configures the plugins
// that could read them in one round. An entry whose value cannot be removed returns to its place
// with its permissions, so the value stays named and the call can be repeated.
func (u *DeleteGroup) Execute(ctx context.Context, group string) error {
	return u.plugins.ApplySecretChange(ctx, func() ([]string, error) {
		var removed []Secret
		var indexes []int
		err := u.ledger.ChangeSecrets(func(list []Secret) ([]Secret, error) {
			removed, indexes = nil, nil
			remaining := make([]Secret, 0, len(list))
			for index, one := range list {
				if one.Group == group {
					removed = append(removed, one)
					indexes = append(indexes, index)
				} else {
					remaining = append(remaining, one)
				}
			}
			if len(removed) == 0 {
				return nil, newNotFoundError(group)
			}
			return remaining, nil
		})
		if err != nil {
			return nil, err
		}
		var failures []error
		for position, one := range removed {
			if err := u.deleteValue(ctx, one.ID); err != nil {
				failures = append(failures, err, u.restoreEntry(one, indexes[position]))
			}
		}
		return nil, errors.Join(failures...)
	})
}

func (u *DeleteGroup) restoreEntry(entry Secret, index int) error {
	return u.ledger.ChangeSecrets(func(list []Secret) ([]Secret, error) {
		if getIndex(list, entry.ID) >= 0 {
			return list, nil
		}
		return slices.Insert(list, min(index, len(list)), entry), nil
	})
}
