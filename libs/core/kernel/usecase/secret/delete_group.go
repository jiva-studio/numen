package secret

import (
	"context"
	"errors"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// DeleteGroup removes every secret of a group with its values and its permissions.
type DeleteGroup struct{ Operation }

// NewDeleteGroup makes the DeleteGroup use case over the operation.
func NewDeleteGroup(op Operation) *DeleteGroup { return &DeleteGroup{Operation: op} }

// Execute removes the entries of the group first and then the values, and configures the plugins
// that could read them in one round. An entry whose value cannot be removed returns to its place
// with its permissions, so the value stays named and the call can be repeated.
func (u *DeleteGroup) Execute(ctx context.Context, group string) error {
	return u.apply(ctx, func() ([]string, error) {
		removed, err := u.removeEntries(group)
		if err != nil {
			return nil, err
		}
		return nil, u.deleteValues(ctx, removed)
	})
}

type removedEntry struct {
	secret plugin.Secret
	index  int
}

func (u *DeleteGroup) removeEntries(group string) ([]removedEntry, error) {
	var removed []removedEntry
	err := u.ledger.ChangeSecrets(func(list []plugin.Secret) ([]plugin.Secret, error) {
		removed = nil
		remaining := make([]plugin.Secret, 0, len(list))
		for index, one := range list {
			if one.Group == group {
				removed = append(removed, removedEntry{secret: one, index: index})
			} else {
				remaining = append(remaining, one)
			}
		}
		if len(removed) == 0 {
			return nil, newNotFoundError(group)
		}
		return remaining, nil
	})
	return removed, err
}

func (u *DeleteGroup) deleteValues(ctx context.Context, removed []removedEntry) error {
	var failures []error
	for _, one := range removed {
		if err := u.deleteValue(ctx, one.secret.ID); err != nil {
			failures = append(failures, err, u.restoreEntry(one.secret, one.index))
		}
	}
	return errors.Join(failures...)
}

func (u *DeleteGroup) restoreEntry(entry plugin.Secret, index int) error {
	return u.ledger.ChangeSecrets(func(list []plugin.Secret) ([]plugin.Secret, error) {
		if getIndex(list, entry.ID) >= 0 {
			return list, nil
		}
		return slices.Insert(list, min(index, len(list)), entry), nil
	})
}
