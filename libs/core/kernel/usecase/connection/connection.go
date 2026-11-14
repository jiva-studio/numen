package connection

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

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

// detectLimit is how long a call waits for the providers of the pending types to answer connections.detected.
const detectLimit = 30 * time.Second

// operation is what every use case of this package is built from.
type operation struct {
	repository connectionport.Repository
	types      connectionport.Types
	keys       pluginport.KeyStore
	uses       connectionport.Uses
	detected   connectionport.Detected
	lists      OptionLists
	gate       *sync.Mutex
}

// OptionLists asks again for the option lists kept for a connection after it changed, and forgets them
// once it is deleted.
type OptionLists interface {
	RefreshConnection(connectionID string)
	DropConnection(pluginID, connectionID string)
}

// Entry is a connection as a window shows it: the fields the type declares, the keys of the secret
// fields that hold a value for the current address, and no value of a secret. IsDetected marks a
// connection the provider of its type reported ready on this machine, which is not stored and cannot be
// changed or deleted. Path is where the provider found a detected connection, empty for a stored one.
type Entry struct {
	ID              string
	TypeID          string
	Name            string
	Revision        int
	Fields          map[string]any
	SecretKeysSet   []string
	IsTypeAvailable bool
	IsComplete      bool
	IsDetected      bool
	Path            string
	Uses            []connection.Use
}

// Deps is what the use cases of this package share: the list, the kinds of connection, the key store and the
// rows of settings that name a connection. Use cases built from one Deps never change the key store and the
// list at once.
type Deps struct{ base operation }

// NewDeps returns the shared parts over the list, the registry of types and the key store. keys is nil on a
// machine with no key store; a connection with no secret field value still works there.
func NewDeps(
	repository connectionport.Repository, types connectionport.Types, keys pluginport.KeyStore, uses connectionport.Uses,
) Deps {
	return Deps{base: operation{
		repository: repository, types: types, keys: keys, uses: uses,
		detected: noDetection{}, lists: noOptionLists{}, gate: &sync.Mutex{},
	}}
}

// noDetection is the detection of a machine where no provider detects connections.
type noDetection struct{}

func (noDetection) ListDetected() []connection.Connection { return nil }
func (noDetection) IsPending(string) bool                 { return false }
func (noDetection) Detect(context.Context, string)        {}
func (noDetection) Refresh()                              {}

type noOptionLists struct{}

func (noOptionLists) RefreshConnection(string)      {}
func (noOptionLists) DropConnection(string, string) {}

// noTitles is the source of titles of a machine where no plugin titles an operation.
type noTitles struct{}

func (noTitles) ListOperationTitles() []manifest.Operation { return nil }

// AddDetection returns the shared parts with the connections the providers detect listed after the stored ones.
func (d Deps) AddDetection(detected connectionport.Detected) Deps {
	d.base.detected = detected
	return d
}

// AddOptionLists returns the shared parts with the option lists an update of a connection refreshes and a
// deletion drops.
func (d Deps) AddOptionLists(lists OptionLists) Deps {
	d.base.lists = lists
	return d
}

// NewCreate returns the use case that adds a connection. newID makes the id of a new connection.
func NewCreate(deps Deps, newID func() (string, error)) *Create {
	return &Create{operation: deps.base, newID: newID}
}

// NewUpdate returns the use case that changes a connection.
func NewUpdate(deps Deps) *Update { return &Update{operation: deps.base} }

// NewDelete returns the use case that removes a connection.
func NewDelete(deps Deps) *Delete { return &Delete{operation: deps.base} }

// NewList returns the use case that lists the connections.
func NewList(deps Deps) *List { return &List{operation: deps.base} }

// NewListTypes returns the use case that lists the kinds of connection and the titles the plugins give
// their operations. titles is nil where no plugin titles an operation.
func NewListTypes(deps Deps, titles connectionport.OperationTitles) *ListTypes {
	if titles == nil {
		titles = noTitles{}
	}
	return &ListTypes{operation: deps.base, titles: titles}
}

// NewResolve returns the use case that finds a connection by its id.
func NewResolve(deps Deps) *Resolve { return &Resolve{operation: deps.base} }

// NewListRows returns the use case that lists the rows a plugin calls an operation through, from the settings
// of rows of the plugins.
func NewListRows(deps Deps, settings connectionport.RowSettings) *ListRows {
	return &ListRows{operation: deps.base, settings: settings}
}

// NewReadSecrets returns the use case that reads the secret fields of a connection.
func NewReadSecrets(deps Deps) *ReadSecrets { return &ReadSecrets{operation: deps.base} }

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

func (o operation) listConnections() ([]connection.Connection, error) {
	list, err := o.repository.ListConnections()
	if err != nil {
		return nil, err
	}
	return append(list, o.listDetectedConnections()...), nil
}

func (o operation) listDetectedConnections() []connection.Connection {
	return o.detected.ListDetected()
}

func (o operation) listConnectionsAfterDetection(typeIDs []string) ([]connection.Connection, error) {
	var plugins []string
	for _, typeID := range typeIDs {
		found, isFound := o.types.GetType(typeID)
		if isFound && o.detected.IsPending(typeID) && !slices.Contains(plugins, found.PluginID) {
			plugins = append(plugins, found.PluginID)
		}
	}
	if len(plugins) > 0 {
		ctx, cancel := context.WithTimeout(plugindomain.WithBackground(context.Background(), plugindomain.WhoDetection), detectLimit)
		defer cancel()
		for _, pluginID := range plugins {
			o.detected.Detect(ctx, pluginID)
		}
	}
	return o.listConnections()
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

func listUses(o operation, one connection.Connection) ([]connection.Use, error) {
	uses, err := o.uses.ListUses(one.ID)
	if err != nil {
		return nil, err
	}
	return addGrantUses(o, uses, one.TypeID), nil
}

func addGrantUses(o operation, uses []connection.Use, typeID string) []connection.Use {
	found, isFound := o.types.GetType(typeID)
	if !isFound || !found.IsAvailable {
		return uses
	}
	for _, granted := range connection.ListGrantUses(o.uses.ListGrants(), found.Declaration) {
		at := slices.IndexFunc(uses, func(one connection.Use) bool { return one.PluginID > granted.PluginID })
		if at < 0 {
			at = len(uses)
		}
		uses = slices.Insert(uses, at, granted)
	}
	return uses
}

func (o operation) buildEntry(ctx context.Context, one connection.Connection) (Entry, error) {
	uses, err := listUses(o, one)
	if err != nil {
		return Entry{}, err
	}
	entry := Entry{
		ID: one.ID, TypeID: one.TypeID, Name: one.Name, Revision: one.Revision,
		Fields: map[string]any{}, Uses: uses, IsDetected: connection.IsDetected(one.ID), Path: one.Path,
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
