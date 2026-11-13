package connection

import (
	"context"
	"errors"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
)

// Delete removes a connection and the secret fields of it from the key store.
type Delete struct{ operation }

// Execute removes the connection from the list first and then its secrets. When a secret cannot be
// removed the connection returns to its place, so the secret stays named and the call can be
// repeated. A connection whose type no plugin declares has no known secret fields, and the entries
// of its secrets stay in the key store. The option lists kept for the connection are dropped with it.
// The rows that name the connection are removed last, and the settings they were in are returned with
// the plugins whose grants reached it, also where the rows cannot be removed.
func (u *Delete) Execute(ctx context.Context, id string) ([]connection.Use, error) {
	u.gate.Lock()
	defer u.gate.Unlock()
	uses, err := u.uses.ListUses(id)
	if err != nil {
		return nil, err
	}
	var removed connection.Connection
	var index int
	err = u.repository.ChangeConnections(func(list []connection.Connection) ([]connection.Connection, error) {
		index = getIndex(list, id)
		if index < 0 {
			return nil, newNotFoundError(id)
		}
		removed = list[index]
		return slices.Delete(slices.Clone(list), index, index+1), nil
	})
	if err != nil {
		return nil, err
	}
	if err := u.removeSecrets(ctx, removed); err != nil {
		return nil, errors.Join(err, u.restoreEntry(removed, index))
	}
	if declared, isFound := u.types.GetType(removed.TypeID); isFound && u.lists != nil {
		u.lists.DropConnection(declared.PluginID, removed.ID)
	}
	uses = addGrantUses(u.operation, uses, removed.TypeID)
	if err := u.uses.RemoveRows(id); err != nil {
		return uses, err
	}
	return uses, nil
}

func (u *Delete) removeSecrets(ctx context.Context, removed connection.Connection) error {
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

func (u *Delete) restoreEntry(entry connection.Connection, index int) error {
	return u.repository.ChangeConnections(func(list []connection.Connection) ([]connection.Connection, error) {
		if getIndex(list, entry.ID) >= 0 {
			return list, nil
		}
		return slices.Insert(list, min(index, len(list)), entry), nil
	})
}
