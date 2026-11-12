package connection

import (
	"context"
	"errors"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
)

// CreateRequest is a new connection. An empty Name makes the core name it from the title of the
// type. Secrets are the values of the secret fields by key.
type CreateRequest struct {
	TypeID  string
	Name    string
	Fields  map[string]any
	Secrets map[string]string
}

// Create adds a connection at the end of the list and keeps its secret fields in the key store.
type Create struct {
	operation
	newID func() (string, error)
}

// Execute stores the secrets, then lists the connection, and returns it. The secrets are stored
// first and removed again where the list cannot be written.
func (u *Create) Execute(ctx context.Context, request CreateRequest) (Entry, error) {
	u.gate.Lock()
	defer u.gate.Unlock()
	declared, err := u.getAvailableType(request.TypeID)
	if err != nil {
		return Entry{}, err
	}
	fields, err := normalizeFields(declared.Declaration, request.Fields)
	if err != nil {
		return Entry{}, err
	}
	if err := checkSecretKeys(declared.Declaration, request.Secrets); err != nil {
		return Entry{}, err
	}
	if err := checkRequiredFields(declared.Declaration, fields, request.Secrets); err != nil {
		return Entry{}, err
	}
	name, err := normalizeName(request.Name)
	if err != nil {
		return Entry{}, err
	}
	hasKey := slices.ContainsFunc(getSecretKeys(declared.Declaration), func(key string) bool { return request.Secrets[key] != "" })
	if err := checkAddress(declared.Declaration, fields, hasKey); err != nil {
		return Entry{}, err
	}
	known, err := u.repository.ListConnections()
	if err != nil {
		return Entry{}, err
	}
	id, err := u.makeID(known)
	if err != nil {
		return Entry{}, err
	}
	if name == "" {
		name = makeName(known, declared.Declaration.Title)
	}
	created := connection.Connection{
		ID: id, TypeID: request.TypeID, Name: name, Revision: 1, Fields: fields,
	}
	if err := u.store(ctx, declared, created, request.Secrets); err != nil {
		return Entry{}, err
	}
	return u.buildEntry(ctx, created)
}

func (u *Create) store(ctx context.Context, declared connection.Type, created connection.Connection, secrets map[string]string) error {
	fingerprint := getFingerprint(created.TypeID, getAddress(declared.Declaration, created.Fields))
	var written []string
	rollback := func(cause error) error {
		var failures []error
		for _, key := range written {
			failures = append(failures, u.secrets().remove(context.WithoutCancel(ctx), created.ID, key))
		}
		return errors.Join(append([]error{cause}, failures...)...)
	}
	for _, key := range getSecretKeys(declared.Declaration) {
		if secrets[key] == "" {
			continue
		}
		if err := u.secrets().set(ctx, created.ID, key, fingerprint, secrets[key]); err != nil {
			return rollback(err)
		}
		written = append(written, key)
	}
	err := u.repository.ChangeConnections(func(list []connection.Connection) ([]connection.Connection, error) {
		if isNameTaken(list, created.Name, "") {
			return nil, newNameTakenError(created.Name)
		}
		if getIndex(list, created.ID) >= 0 {
			return nil, newInvalidError("the id %s is used", created.ID)
		}
		return append(slices.Clone(list), created), nil
	})
	if err != nil {
		return rollback(err)
	}
	return nil
}

func (u *Create) makeID(list []connection.Connection) (string, error) {
	for {
		id, err := u.newID()
		if err != nil || getIndex(list, id) < 0 {
			return id, err
		}
	}
}
