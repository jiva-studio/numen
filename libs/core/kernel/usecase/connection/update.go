package connection

import (
	"context"
	"errors"
	"maps"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// UpdateRequest changes one connection. Revision is the revision the caller read. Fields are the
// values of the editable fields; a key left out is cleared. A key left out of Secrets keeps its
// value and an empty value clears it.
type UpdateRequest struct {
	ID       string
	Revision int
	Name     string
	Fields   map[string]any
	Secrets  map[string]string
}

// Update changes the name, the fields and the secrets of a stored connection, and adds 1 to its revision. A
// detected connection is not found.
type Update struct{ operation }

// Execute applies the request to the connection that has the revision the request carries. A change
// of the address clears every secret field before the secrets of the request are stored. Fields the
// type no longer declares stay as they are.
func (u *Update) Execute(ctx context.Context, request UpdateRequest) (Entry, error) {
	u.gate.Lock()
	defer u.gate.Unlock()
	list, err := u.listConnections()
	if err != nil {
		return Entry{}, err
	}
	current, err := findCurrent(list, request)
	if err != nil {
		return Entry{}, err
	}
	declared, err := u.getAvailableType(current.TypeID)
	if err != nil {
		return Entry{}, err
	}
	updated, plan, err := u.plan(ctx, declared, current, request)
	if err != nil {
		return Entry{}, err
	}
	if isNameTaken(list, updated.Name, updated.ID) {
		return Entry{}, newNameTakenError(updated.Name)
	}
	backup, err := u.secrets().take(ctx, current.ID, plan.touched())
	if err != nil {
		return Entry{}, err
	}
	if err := u.apply(ctx, current.ID, plan); err != nil {
		return Entry{}, errors.Join(err, u.secrets().restore(context.WithoutCancel(ctx), current.ID, backup))
	}
	err = u.repository.ChangeConnections(func(list []connection.Connection) ([]connection.Connection, error) {
		position := getIndex(list, current.ID)
		switch {
		case position < 0:
			return nil, newNotFoundError(current.ID)
		case list[position].Revision != current.Revision:
			return nil, newStaleRevisionError(current.ID)
		case isNameTaken(append(slices.Clone(list), u.detected.ListDetected()...), updated.Name, updated.ID):
			return nil, newNameTakenError(updated.Name)
		}
		changed := slices.Clone(list)
		changed[position] = updated
		return changed, nil
	})
	if err != nil {
		return Entry{}, errors.Join(err, u.secrets().restore(context.WithoutCancel(ctx), current.ID, backup))
	}
	return u.describe(ctx, updated)
}

func (u *Update) describe(ctx context.Context, updated connection.Connection) (Entry, error) {
	entry, err := u.buildEntry(ctx, updated)
	if err != nil {
		return Entry{}, err
	}
	return entry, u.uses.Refresh(ctx, updated.ID)
}

func findCurrent(list []connection.Connection, request UpdateRequest) (connection.Connection, error) {
	index := getIndex(list, request.ID)
	if index < 0 || connection.IsDetected(request.ID) {
		return connection.Connection{}, newNotFoundError(request.ID)
	}
	if list[index].Revision != request.Revision {
		return connection.Connection{}, newStaleRevisionError(request.ID)
	}
	return list[index], nil
}

// secretPlan is what an update does to the key store: the keys to clear and the values to store.
type secretPlan struct {
	fingerprint string
	clear       []string
	store       map[string]string
}

func (p secretPlan) touched() []string {
	keys := slices.Clone(p.clear)
	for key := range p.store {
		if !slices.Contains(keys, key) {
			keys = append(keys, key)
		}
	}
	return keys
}

func (u *Update) plan(
	ctx context.Context, declared connection.Type, current connection.Connection, request UpdateRequest,
) (connection.Connection, secretPlan, error) {
	edited, err := normalizeFields(declared.Declaration, request.Fields)
	if err != nil {
		return connection.Connection{}, secretPlan{}, err
	}
	if err := checkSecretKeys(declared.Declaration, request.Secrets); err != nil {
		return connection.Connection{}, secretPlan{}, err
	}
	name, err := normalizeName(request.Name)
	if err != nil {
		return connection.Connection{}, secretPlan{}, err
	}
	if name == "" {
		name = current.Name
	}
	updated := current
	updated.Name = name
	updated.Revision = current.Revision + 1
	updated.Fields = mergeFields(declared.Declaration, current.Fields, edited)

	plan, hasKey, err := u.planSecrets(ctx, declared, current, updated, request.Secrets)
	if err != nil {
		return connection.Connection{}, secretPlan{}, err
	}
	if err := checkAddress(declared.Declaration, updated.Fields, hasKey); err != nil {
		return connection.Connection{}, secretPlan{}, err
	}
	return updated, plan, nil
}

func (u *Update) planSecrets(
	ctx context.Context, declared connection.Type, current, updated connection.Connection, requested map[string]string,
) (secretPlan, bool, error) {
	oldAddress := getAddress(declared.Declaration, current.Fields)
	newAddress := getAddress(declared.Declaration, updated.Fields)
	plan := secretPlan{fingerprint: getFingerprint(current.TypeID, newAddress), store: map[string]string{}}
	oldFingerprint := getFingerprint(current.TypeID, oldAddress)
	hasKey := false
	for _, key := range getSecretKeys(declared.Declaration) {
		value, isNamed := requested[key]
		isKept := false
		if !isNamed && oldAddress == newAddress {
			isSet, err := u.secrets().isSet(ctx, current.ID, key, oldFingerprint)
			if err != nil {
				return secretPlan{}, false, err
			}
			isKept = isSet
		}
		switch {
		case isNamed && value != "":
			plan.store[key] = value
			hasKey = true
		case isKept:
			hasKey = true
		case isNamed || oldAddress != newAddress:
			plan.clear = append(plan.clear, key)
		}
	}
	return plan, hasKey, nil
}

func (u *Update) apply(ctx context.Context, id string, plan secretPlan) error {
	for _, key := range plan.clear {
		if _, isStored := plan.store[key]; isStored {
			continue
		}
		if err := u.secrets().remove(ctx, id, key); err != nil {
			return err
		}
	}
	for _, key := range slices.Sorted(maps.Keys(plan.store)) {
		if err := u.secrets().set(ctx, id, key, plan.fingerprint, plan.store[key]); err != nil {
			return err
		}
	}
	return nil
}

func mergeFields(declared manifest.ConnectionType, stored, edited map[string]any) map[string]any {
	merged := map[string]any{}
	for key, value := range stored {
		if _, isDeclared := declared.GetField(key); !isDeclared {
			merged[key] = value
		}
	}
	maps.Copy(merged, edited)
	return merged
}
