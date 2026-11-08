package connection

import (
	"context"
	"errors"
	"slices"

	"github.com/jiva-studio/numen/libs/core/port/connectionport"
)

// Delete removes a connection and the secret fields of it from the key store.
type Delete struct{ operation }

// Execute removes the connection from the list first and then its secrets. When a secret cannot be
// removed the connection returns to its place, so the secret stays named and the call can be
// repeated. A connection whose type no plugin declares has no known secret fields, and the entries
// of its secrets stay in the key store.
func (u *Delete) Execute(ctx context.Context, id string) error {
	u.gate.Lock()
	defer u.gate.Unlock()
	var removed connectionport.Connection
	var index int
	err := u.repository.ChangeConnections(func(list []connectionport.Connection) ([]connectionport.Connection, error) {
		index = getIndex(list, id)
		if index < 0 {
			return nil, newNotFoundError(id)
		}
		removed = list[index]
		return slices.Delete(slices.Clone(list), index, index+1), nil
	})
	if err != nil {
		return err
	}
	if err := u.removeSecrets(ctx, removed); err != nil {
		return errors.Join(err, u.restoreEntry(removed, index))
	}
	return nil
}

func (u *Delete) removeSecrets(ctx context.Context, removed connectionport.Connection) error {
	declared, isFound := u.types.GetType(removed.TypeID)
	if !isFound {
		return nil
	}
	var failures []error
	for _, key := range getSecretKeys(declared.Declaration) {
		failures = append(failures, u.secrets().remove(ctx, removed.ID, key))
	}
	return errors.Join(failures...)
}

func (u *Delete) restoreEntry(entry connectionport.Connection, index int) error {
	return u.repository.ChangeConnections(func(list []connectionport.Connection) ([]connectionport.Connection, error) {
		if getIndex(list, entry.ID) >= 0 {
			return list, nil
		}
		return slices.Insert(list, min(index, len(list)), entry), nil
	})
}
