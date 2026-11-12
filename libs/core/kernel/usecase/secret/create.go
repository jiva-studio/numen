package secret

import (
	"context"
	"errors"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// Create adds a secret at the end of the list, which is the end of its group, and keeps its value in the key store.
type Create struct {
	Operation
	newID func() (string, error)
}

func NewCreate(op Operation, newID func() (string, error)) *Create {
	return &Create{Operation: op, newID: newID}
}

// Execute stores the value, then lists the secret, and returns its id. The value is stored first,
// and removed again where the list cannot be written.
func (u *Create) Execute(ctx context.Context, group, name, value string) (string, error) {
	if err := checkNewSecret(group, name, value); err != nil {
		return "", err
	}
	var id string
	err := u.apply(ctx, func() ([]string, error) {
		known, err := u.ledger.GetSecrets()
		if err != nil {
			return nil, err
		}
		if isNameTaken(known, group, name, "") {
			return nil, newInvalidError("the name is taken in the group")
		}
		if id, err = u.makeID(known); err != nil {
			return nil, err
		}
		if err := u.setValue(ctx, id, value); err != nil {
			return nil, err
		}
		change := func(list []plugin.Secret) ([]plugin.Secret, error) {
			if isNameTaken(list, group, name, "") {
				return nil, newInvalidError("the name is taken in the group")
			}
			return append(slices.Clone(list), plugin.Secret{ID: id, Name: name, Group: group}), nil
		}
		if err := u.ledger.ChangeSecrets(change); err != nil {
			return nil, errors.Join(err, u.deleteValue(context.WithoutCancel(ctx), id))
		}
		return nil, nil
	})
	if err != nil {
		return "", err
	}
	return id, nil
}

func (u *Create) makeID(list []plugin.Secret) (string, error) {
	for {
		id, err := u.newID()
		if err != nil || getIndex(list, id) < 0 {
			return id, err
		}
	}
}

func checkNewSecret(group, name, value string) error {
	if err := checkName("group", group); err != nil {
		return err
	}
	if err := checkName("name", name); err != nil {
		return err
	}
	if value == "" {
		return newInvalidError("the value is empty")
	}
	return nil
}
