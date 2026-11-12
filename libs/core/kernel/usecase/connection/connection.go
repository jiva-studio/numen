package connection

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

var (
	// ErrInvalidField is a request whose name, field, secret or address the type does not accept.
	ErrInvalidField = errors.New("invalid connection field")
	// ErrNameTaken is a name that another connection has, ignoring case.
	ErrNameTaken = errors.New("the connection name is taken")
)

// operation is what every use case of this package is built from.
type operation struct {
	repository connectionport.Repository
	types      connectionport.Types
	keys       pluginport.KeyStore
	uses       connectionport.Uses
	gate       *sync.Mutex
}

// Entry is a connection as a window shows it: the fields the type declares, the keys of the secret
// fields that hold a value for the current address, and no value of a secret.
type Entry struct {
	ID                      string
	TypeID                  string
	Name                    string
	Revision                int
	Fields                  map[string]any
	SecretKeysSet           []string
	IsTypeAvailable         bool
	IsComplete              bool
	IsPrivateNetworkAllowed bool
	Uses                    []connection.Use
}

// Connections holds every use case on the list of connections.
type Connections struct {
	Create      *Create
	Update      *Update
	Delete      *Delete
	List        *List
	ListTypes   *ListTypes
	Resolve     *Resolve
	ReadSecrets *ReadSecrets
}

// NewConnections returns the use cases over the list, the registry of types and the key store.
// newID makes the id of a new connection. keys is nil on a machine with no key store; a connection
// with no secret field value still works there. The use cases never change the key store and the
// list at once.
func NewConnections(
	repository connectionport.Repository, types connectionport.Types, keys pluginport.KeyStore, uses connectionport.Uses, newID func() (string, error),
) *Connections {
	base := operation{repository: repository, types: types, keys: keys, uses: uses, gate: &sync.Mutex{}}
	return &Connections{
		Create:      &Create{operation: base, newID: newID},
		Update:      &Update{operation: base},
		Delete:      &Delete{operation: base},
		List:        &List{operation: base},
		ListTypes:   &ListTypes{operation: base},
		Resolve:     &Resolve{operation: base},
		ReadSecrets: &ReadSecrets{operation: base},
	}
}

func newInvalidError(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidField, fmt.Sprintf(format, args...))
}

func newNotFoundError(id string) error { return fmt.Errorf("%w: %s", connection.ErrNotFound, id) }

func newNameTakenError(name string) error { return fmt.Errorf("%w: %s", ErrNameTaken, name) }

func newStaleRevisionError(id string) error {
	return fmt.Errorf("%w: %s", connection.ErrStaleRevision, id)
}

func getIndex(list []connection.Connection, id string) int {
	return slices.IndexFunc(list, func(one connection.Connection) bool { return one.ID == id })
}

func isNameTaken(list []connection.Connection, name, exceptID string) bool {
	return slices.ContainsFunc(list, func(one connection.Connection) bool {
		return one.ID != exceptID && strings.EqualFold(one.Name, name)
	})
}

func makeName(list []connection.Connection, title string) string {
	base := truncateName(title, connection.MaxNameLength-len(" 99999"))
	if !isNameTaken(list, base, "") {
		return base
	}
	for number := 2; ; number++ {
		name := fmt.Sprintf("%s %d", base, number)
		if !isNameTaken(list, name, "") {
			return name
		}
	}
}

func truncateName(name string, limit int) string {
	runes := []rune(name)
	if len(runes) <= limit {
		return name
	}
	return strings.TrimSpace(string(runes[:limit]))
}

func (o operation) getAvailableType(typeID string) (connection.Type, error) {
	found, isFound := o.types.GetType(typeID)
	if !isFound || !found.IsAvailable {
		return connection.Type{}, fmt.Errorf("%w: %s", connection.ErrTypeUnavailable, typeID)
	}
	return found, nil
}

func newKeyStoreError(action string, err error) error {
	return fmt.Errorf("%w: %s: %w", connection.ErrKeyStore, action, err)
}

// secretStore reads and writes the secret fields of one connection in the key store.
type secretStore struct {
	keys pluginport.KeyStore
}

func (o operation) secrets() secretStore { return secretStore{keys: o.keys} }

func (s secretStore) set(ctx context.Context, id, key, fingerprint, value string) error {
	if s.keys == nil {
		return newKeyStoreError("store "+key, plugindomain.ErrNoKeyStore)
	}
	if err := s.keys.SetKey(ctx, GetEntryName(id, key), encodeSecret(fingerprint, value)); err != nil {
		return newKeyStoreError("store "+key, err)
	}
	return nil
}

func (s secretStore) remove(ctx context.Context, id, key string) error {
	if s.keys == nil {
		return nil
	}
	if err := s.keys.DeleteKey(ctx, GetEntryName(id, key)); err != nil && !errors.Is(err, plugindomain.ErrNoKey) {
		return newKeyStoreError("delete "+key, err)
	}
	return nil
}

func (s secretStore) read(ctx context.Context, id, key string) (entry string, isFound bool, err error) {
	if s.keys == nil {
		return "", false, nil
	}
	entry, err = s.keys.GetKey(ctx, GetEntryName(id, key))
	if errors.Is(err, plugindomain.ErrNoKey) {
		return "", false, nil
	}
	if err != nil {
		return "", false, newKeyStoreError("read "+key, err)
	}
	return entry, true, nil
}

func (s secretStore) isSet(ctx context.Context, id, key, fingerprint string) (bool, error) {
	entry, isFound, err := s.read(ctx, id, key)
	if err != nil || !isFound {
		return false, err
	}
	return hasFingerprint(entry, fingerprint), nil
}

// snapshot is the content of key store entries at one moment, kept to put them back.
type snapshot map[string]*string

func (s secretStore) take(ctx context.Context, id string, keys []string) (snapshot, error) {
	backup := snapshot{}
	for _, key := range keys {
		entry, isFound, err := s.read(ctx, id, key)
		if err != nil {
			return nil, err
		}
		if isFound {
			backup[key] = &entry
		} else {
			backup[key] = nil
		}
	}
	return backup, nil
}

func (s secretStore) restore(ctx context.Context, id string, backup snapshot) error {
	if s.keys == nil {
		return nil
	}
	var failures []error
	for key, entry := range backup {
		if entry == nil {
			failures = append(failures, s.remove(ctx, id, key))
			continue
		}
		if err := s.keys.SetKey(ctx, GetEntryName(id, key), *entry); err != nil {
			failures = append(failures, newKeyStoreError("restore "+key, err))
		}
	}
	return errors.Join(failures...)
}

func (o operation) buildEntry(ctx context.Context, one connection.Connection) (Entry, error) {
	uses, err := o.uses.ListUses(one.ID)
	if err != nil {
		return Entry{}, err
	}
	entry := Entry{
		ID: one.ID, TypeID: one.TypeID, Name: one.Name, Revision: one.Revision,
		Fields: map[string]any{}, IsPrivateNetworkAllowed: one.IsPrivateNetworkAllowed, Uses: uses,
	}
	found, isFound := o.types.GetType(one.TypeID)
	if !isFound {
		maps.Copy(entry.Fields, one.Fields)
		return entry, nil
	}
	entry.IsTypeAvailable = found.IsAvailable
	entry.IsComplete = true
	fingerprint := getFingerprint(one.TypeID, getAddress(found.Declaration, one.Fields))
	for _, field := range found.Declaration.Fields {
		isSet, err := o.addField(ctx, &entry, one, field, fingerprint)
		if err != nil {
			return Entry{}, err
		}
		entry.IsComplete = entry.IsComplete && (isSet || !field.IsRequired || field.FixedValue != nil)
	}
	return entry, nil
}

func (o operation) addField(
	ctx context.Context, entry *Entry, one connection.Connection, field manifest.ConnectionField, fingerprint string,
) (bool, error) {
	if field.Type == secretFieldType {
		isSet, err := o.secrets().isSet(ctx, one.ID, field.Key, fingerprint)
		if isSet {
			entry.SecretKeysSet = append(entry.SecretKeysSet, field.Key)
		}
		return isSet, err
	}
	value, isSet := one.Fields[field.Key]
	if isSet && field.FixedValue == nil {
		entry.Fields[field.Key] = value
	}
	return isSet, nil
}
