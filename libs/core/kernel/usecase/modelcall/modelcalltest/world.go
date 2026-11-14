package modelcalltest

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/connectiontest"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	connectionuse "github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"
)

// The plugins and the types of the fake provider.
const (
	ProviderPluginID  = "md.example.provider"
	OtherPluginID     = "md.example.other"
	ServerTypeID      = ProviderPluginID + "/server"
	CompleterTypeID   = ProviderPluginID + "/completer"
	OtherServerTypeID = OtherPluginID + "/server"
	LimitedTypeID     = ProviderPluginID + "/limited"
	AgentTypeID       = ProviderPluginID + "/agent"
)

// Notices records the reports.
type Notices struct {
	mu   sync.Mutex
	list []connection.SetAside
}

func (n *Notices) ReportSetAside(setAside connection.SetAside) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.list = append(n.list, setAside)
}

// List returns the reports, in order.
func (n *Notices) List() []connection.SetAside {
	n.mu.Lock()
	defer n.mu.Unlock()
	return slices.Clone(n.list)
}

// Rows holds the rows of each plugin by operation.
type Rows struct {
	mu   sync.Mutex
	rows map[string][]connection.Row
	Err  error
}

func (r *Rows) Set(pluginID, operation string, rows ...connection.Row) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.rows == nil {
		r.rows = map[string][]connection.Row{}
	}
	r.rows[pluginID+"\x00"+operation] = rows
}

func (r *Rows) ListRows(pluginID, operation string) ([]connection.Row, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.rows[pluginID+"\x00"+operation]), r.Err
}

// Source opens the fake provider for any plugin and counts the leases.
type Source struct {
	mu       sync.Mutex
	provider *Provider
	opened   []string
	released int
	failOpen map[string]error
}

func NewSource(provider *Provider) *Source {
	return &Source{provider: provider, failOpen: map[string]error{}}
}

// FailOpen makes the plugin fail to start.
func (s *Source) FailOpen(pluginID string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failOpen[pluginID] = err
}

// OpenProvider records the plugin and returns the provider.
func (s *Source) OpenProvider(_ context.Context, pluginID string) (connectionport.Provider, func(), error) {
	release, err := s.open(pluginID)
	if err != nil {
		return nil, nil, err
	}
	return s.provider, release, nil
}

func (s *Source) open(pluginID string) (func(), error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.failOpen[pluginID]; err != nil {
		return nil, err
	}
	s.opened = append(s.opened, pluginID)
	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.released++
	}, nil
}

// Opened returns the plugin ids opened, in order.
func (s *Source) Opened() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.opened)
}

// Released returns how many leases were released.
func (s *Source) Released() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.released
}

type memoryRepository struct {
	mu   sync.Mutex
	list []connection.Connection
}

func (r *memoryRepository) ListConnections() ([]connection.Connection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.list), nil
}

func (r *memoryRepository) ChangeConnections(change func([]connection.Connection) ([]connection.Connection, error)) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	updated, err := change(slices.Clone(r.list))
	if err != nil {
		return err
	}
	r.list = updated
	return nil
}

type fixedTypes struct{ list []connection.Type }

func (t fixedTypes) ListTypes() []connection.Type { return t.list }

func (t fixedTypes) GetType(typeID string) (connection.Type, bool) {
	for _, one := range t.list {
		if one.GetID() == typeID {
			return one, true
		}
	}
	return connection.Type{}, false
}

// Keys is a key store in memory. OnGet runs before every read.
type Keys struct {
	mu     sync.Mutex
	Values map[string]string
	OnGet  func()
}

// GetKey returns the key stored under the name.
func (k *Keys) GetKey(_ context.Context, name string) (string, error) {
	if k.OnGet != nil {
		k.OnGet()
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	value, found := k.Values[name]
	if !found {
		return "", plugindomain.ErrNoKey
	}
	return value, nil
}

func (k *Keys) SetKey(_ context.Context, name, key string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.Values[name] = key
	return nil
}

// DeleteKey removes the key stored under the name.
func (k *Keys) DeleteKey(_ context.Context, name string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if _, found := k.Values[name]; !found {
		return plugindomain.ErrNoKey
	}
	delete(k.Values, name)
	return nil
}

// World is the connections, the key store, the rows and the fake provider a call goes through.
type World struct {
	Connections *connectiontest.Connections
	Resolve     connectionport.ResolveConnection
	ReadSecrets connectionport.ReadConnectionSecrets
	Keys        *Keys
	Rows        *Rows
	Provider    *Provider
	Source      *Source
	Notices     *Notices
	Waits       *Waits
	Clock       *Clock
	Types       *fixedTypes
	Detected    *Detected
	repository  *memoryRepository
	next        int
}

// New creates a world with two plugins that provide the type server (llm.complete, llm.embed, llm.chat
// and the list llm.models of its row field model) and, for the first plugin, completer (llm.complete only).
func New() *World {
	address := manifest.ConnectionField{Key: addressField, Type: textField, Title: "Address", IsAddress: true, IsRequired: true}
	key := manifest.ConnectionField{Key: apiKeyField, Type: "secret", Title: "API key", IsRequired: true}
	minimum, step := 1.0, 1.0
	parallelRequests := manifest.ConnectionField{Key: manifest.ConnectionParallelRequests, Type: "number", Title: "Parallel requests", Default: float64(2), Min: &minimum, Step: &step}
	optionalKey := manifest.ConnectionField{Key: apiKeyField, Type: "secret", Title: "API key"}
	modelField := manifest.ConnectionField{Key: ModelOption, Type: textField, Title: "Model", IsRequired: true, OptionsFrom: ModelsOperation}
	maximum := 8.0
	agentParallelRequests := manifest.ConnectionField{Key: manifest.ConnectionParallelRequests, Type: "number", Title: "Parallel requests", Default: float64(1), Min: &minimum, Max: &maximum, Step: &step}
	serverType := func(pluginID string) connection.Type {
		return connection.Type{PluginID: pluginID, PluginName: pluginID, IsAvailable: true, Declaration: manifest.ConnectionType{
			ID: "server", Title: "Server", Operations: getBothOperations(), Fields: []manifest.ConnectionField{address, key},
			RowFields: []manifest.ConnectionField{modelField},
		}}
	}
	types := &fixedTypes{list: []connection.Type{
		serverType(ProviderPluginID),
		{PluginID: ProviderPluginID, PluginName: ProviderPluginID, IsAvailable: true, Declaration: manifest.ConnectionType{
			ID: "completer", Title: "Completer", Operations: getCompleteOnly(), Fields: []manifest.ConnectionField{address, optionalKey},
		}},
		serverType(OtherPluginID),
		{PluginID: ProviderPluginID, PluginName: ProviderPluginID, IsAvailable: true, Declaration: manifest.ConnectionType{
			ID: "limited", Title: "Limited", Operations: getBothOperations(), Fields: []manifest.ConnectionField{address, key, parallelRequests},
		}},
		{PluginID: ProviderPluginID, PluginName: ProviderPluginID, IsAvailable: true, Declaration: manifest.ConnectionType{
			ID: "agent", Title: "Agent", Operations: getCompleteOnly(), Fields: []manifest.ConnectionField{agentParallelRequests},
		}},
	}}
	w := &World{
		Keys: &Keys{Values: map[string]string{}}, Rows: &Rows{}, Provider: NewProvider(), Notices: &Notices{}, Waits: newWaits(),
		Clock: &Clock{now: time.Date(2026, time.March, 2, 12, 0, 0, 0, time.UTC)}, Types: types, repository: &memoryRepository{},
	}
	w.Source = NewSource(w.Provider)
	w.Detected = &Detected{}
	w.Connections = connectiontest.NewWithDetection(w.repository, types, w.Keys, NoUses{}, w.Detected, func() (string, error) {
		w.next++
		return fmt.Sprintf("c%08x", w.next), nil
	})
	w.Resolve, w.ReadSecrets = w.Connections.Resolve.Execute, w.Connections.ReadSecrets.Execute
	return w
}

// Add creates a connection of the type with the address and, when key is not empty, the key.
func (w *World) Add(t *testing.T, typeID, name, address, key string) Entry {
	t.Helper()
	request := connectionuse.CreateRequest{TypeID: typeID, Name: name, Fields: map[string]any{addressField: address}}
	if key != "" {
		request.Secrets = map[string]string{apiKeyField: key}
	}
	entry, err := w.Connections.Create.Execute(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	return newEntry(entry)
}

// AddServer creates a connection of the type server of the first plugin.
func (w *World) AddServer(t *testing.T, name, address, key string) Entry {
	t.Helper()
	return w.Add(t, ServerTypeID, name, address, key)
}

// Edit changes the name of the connection, which adds 1 to its revision.
func (w *World) Edit(t *testing.T, entry Entry, name string) Entry {
	t.Helper()
	changed, err := w.Connections.Update.Execute(t.Context(), connectionuse.UpdateRequest{
		ID: entry.ID, Revision: entry.Revision, Name: name, Fields: entry.Fields,
	})
	if err != nil {
		t.Fatal(err)
	}
	return newEntry(changed)
}

// ChangeRevision adds 1 to the revision of the connection with no other change.
func (w *World) ChangeRevision(t *testing.T, connectionID string) {
	t.Helper()
	err := w.repository.ChangeConnections(func(list []connection.Connection) ([]connection.Connection, error) {
		for i := range list {
			if list[i].ID == connectionID {
				list[i].Revision++
			}
		}
		return list, nil
	})
	if err != nil {
		t.Error(err)
	}
}

// Row is a row that names the connection with the model as its option.
func Row(entry Entry, model string) connection.Row {
	return connection.Row{ConnectionID: entry.ID, Options: map[string]string{ModelOption: model}}
}

// Clock is a clock that stands still until Advance moves it.
type Clock struct {
	mu  sync.Mutex
	now time.Time
}

// Now returns the time of the clock.
func (c *Clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Advance moves the clock forward.
func (c *Clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// SetAvailable marks the types of the plugin as provided by an enabled plugin, or not.
func (w *World) SetAvailable(pluginID string, isAvailable bool) {
	for i := range w.Types.list {
		if w.Types.list[i].PluginID == pluginID {
			w.Types.list[i].IsAvailable = isAvailable
		}
	}
}

const apiKeyField = "apiKey"

// NoUses is the settings of rows of an installation where no plugin has one.
type NoUses struct{}

func (NoUses) ListUses(string) ([]connection.Use, error) { return nil, nil }

// RemoveRows removes nothing.
func (NoUses) RemoveRows(string) error { return nil }

// Refresh configures no plugin.
func (NoUses) Refresh(context.Context, string) error { return nil }

func (NoUses) ListGrants() []connection.Grant { return nil }

// AddLimited creates a connection of the type limited, which runs two calls at once unless the field
// parallelRequests says otherwise. A parallelRequests of 0 leaves the field unset.
func (w *World) AddLimited(t *testing.T, name, address string, parallelRequests int) Entry {
	t.Helper()
	fields := map[string]any{addressField: address}
	if parallelRequests > 0 {
		fields[manifest.ConnectionParallelRequests] = float64(parallelRequests)
	}
	entry, err := w.Connections.Create.Execute(t.Context(), connectionuse.CreateRequest{
		TypeID: LimitedTypeID, Name: name, Fields: fields, Secrets: map[string]string{apiKeyField: "sk-" + name},
	})
	if err != nil {
		t.Fatal(err)
	}
	return newEntry(entry)
}

const addressField = "baseUrl"

const textField = "string"

// The operations of the fake provider, the list of models of its row field and the key of that field.
const (
	CompleteOperation = "llm.complete"
	EmbedOperation    = "llm.embed"
	ChatOperation     = "llm.chat"
	ModelsOperation   = "llm.models"
	ModelOption       = "model"
)

func getBothOperations() []string {
	return []string{CompleteOperation, EmbedOperation, ChatOperation, ModelsOperation}
}

func getCompleteOnly() []string { return []string{CompleteOperation} }

// AddAgent creates a connection of the type agent, which has no address and no key and runs one call at a
// time.
func (w *World) AddAgent(t *testing.T, name string) Entry {
	t.Helper()
	entry, err := w.Connections.Create.Execute(t.Context(), connectionuse.CreateRequest{
		TypeID: AgentTypeID, Name: name, Fields: map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return newEntry(entry)
}

// Entry is a connection the world created: its id, name, revision and stored fields.
type Entry struct {
	ID       string
	Name     string
	Revision int
	Fields   map[string]any
}

func newEntry(created connectionuse.Entry) Entry {
	return Entry{ID: created.ID, Name: created.Name, Revision: created.Revision, Fields: created.Fields}
}

// AddDetected adds a type of the first plugin whose calls go to the address, and the connection its provider
// detects, named name.
func (w *World) AddDetected(name, address string) Entry {
	declared := manifest.ConnectionType{
		ID: "detecting", Title: "Detecting", Operations: append(getCompleteOnly(), connection.OperationDetected),
		Fields: []manifest.ConnectionField{{Key: addressField, Type: textField, Title: "Address", FixedValue: &address}},
	}
	w.Types.list = append(w.Types.list, connection.Type{PluginID: ProviderPluginID, PluginName: ProviderPluginID, IsAvailable: true, Declaration: declared})
	typeID := manifest.GetConnectionTypeID(ProviderPluginID, declared.ID)
	detected := connection.Connection{ID: connection.GetDetectedID(typeID, name), TypeID: typeID, Name: name, Fields: map[string]any{}}
	w.Detected.add(detected)
	return Entry{ID: detected.ID, Name: name, Fields: detected.Fields}
}

// Detected is the connections the providers of the world detected, every type answered.
type Detected struct {
	mu   sync.Mutex
	list []connection.Connection
}

func (d *Detected) add(one connection.Connection) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.list = append(d.list, one)
}

// ListDetected returns the connections added, in order.
func (d *Detected) ListDetected() []connection.Connection {
	d.mu.Lock()
	defer d.mu.Unlock()
	return slices.Clone(d.list)
}

// IsPending reports none: every type answered.
func (d *Detected) IsPending(string) bool { return false }

// Detect asks nothing.
func (d *Detected) Detect(context.Context, string) {}

// Refresh asks nothing.
func (d *Detected) Refresh() {}

// AddRowField appends the field to the type with typeID; a declaration read before keeps the row fields it
// had.
func (w *World) AddRowField(typeID string, field manifest.ConnectionField) {
	for i := range w.Types.list {
		if w.Types.list[i].GetID() == typeID {
			w.Types.list[i].Declaration.RowFields = append(slices.Clone(w.Types.list[i].Declaration.RowFields), field)
		}
	}
}

// SetPluginVersion sets the version of the plugin that provides the types of the plugin.
func (w *World) SetPluginVersion(pluginID, version string) {
	for i := range w.Types.list {
		if w.Types.list[i].PluginID == pluginID {
			w.Types.list[i].PluginVersion = version
		}
	}
}
