package modelcalltest

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/usecase/connection"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
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

// Notice is one report of a connection set aside.
type Notice struct {
	ConnectionID   string
	ConnectionName string
	Reason         connectionport.FailureReason
}

// Notices records the reports.
type Notices struct {
	mu   sync.Mutex
	list []Notice
}

// ReportSetAside records the report.
func (n *Notices) ReportSetAside(connectionID, connectionName string, reason connectionport.FailureReason) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.list = append(n.list, Notice{ConnectionID: connectionID, ConnectionName: connectionName, Reason: reason})
}

// List returns the reports, in order.
func (n *Notices) List() []Notice {
	n.mu.Lock()
	defer n.mu.Unlock()
	return slices.Clone(n.list)
}

// Rows holds the rows of each plugin by operation.
type Rows struct {
	mu   sync.Mutex
	rows map[string][]connectionport.Row
	Err  error
}

// Set replaces the rows of the plugin for the operation.
func (r *Rows) Set(pluginID, operation string, rows ...connectionport.Row) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.rows == nil {
		r.rows = map[string][]connectionport.Row{}
	}
	r.rows[pluginID+"\x00"+operation] = rows
}

// ListRows returns the rows of the plugin for the operation.
func (r *Rows) ListRows(pluginID, operation string) ([]connectionport.Row, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.rows[pluginID+"\x00"+operation]), r.Err
}

// Source opens the fake provider for any plugin and counts the leases.
type Source struct {
	mu       sync.Mutex
	provider connectionport.Provider
	opened   []string
	released int
	failOpen map[string]error
}

// NewSource creates a source that opens provider.
func NewSource(provider connectionport.Provider) *Source {
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
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.failOpen[pluginID]; err != nil {
		return nil, nil, err
	}
	s.opened = append(s.opened, pluginID)
	return s.provider, func() {
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
	list []connectionport.Connection
}

func (r *memoryRepository) ListConnections() ([]connectionport.Connection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.list), nil
}

func (r *memoryRepository) ChangeConnections(change func([]connectionport.Connection) ([]connectionport.Connection, error)) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	updated, err := change(slices.Clone(r.list))
	if err != nil {
		return err
	}
	r.list = updated
	return nil
}

type fixedTypes struct{ list []connectionport.Type }

func (t fixedTypes) ListTypes() []connectionport.Type { return t.list }

func (t fixedTypes) GetType(typeID string) (connectionport.Type, bool) {
	for _, one := range t.list {
		if one.GetID() == typeID {
			return one, true
		}
	}
	return connectionport.Type{}, false
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
		return "", pluginport.ErrNoKey
	}
	return value, nil
}

// SetKey stores the key under the name.
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
		return pluginport.ErrNoKey
	}
	delete(k.Values, name)
	return nil
}

// World is the connections, the key store, the rows and the fake provider a call goes through.
type World struct {
	Connections *connection.Connections
	Keys        *Keys
	Rows        *Rows
	Provider    *Provider
	Source      *Source
	Notices     *Notices
	Waits       *Waits
	Clock       *Clock
	Types       *fixedTypes
	repository  *memoryRepository
	next        int
}

// New creates a world with two plugins that provide the types server (complete and embed) and, for the
// first plugin, completer (complete only).
func New() *World {
	address := manifest.ConnectionField{Key: addressField, Type: "string", Title: "Address", IsAddress: true, IsRequired: true}
	key := manifest.ConnectionField{Key: apiKeyField, Type: "secret", Title: "API key", IsRequired: true}
	minimum, step := 1.0, 1.0
	parallelRequests := manifest.ConnectionField{Key: manifest.ConnectionParallelRequests, Type: "number", Title: "Parallel requests", Default: float64(2), Min: &minimum, Step: &step}
	optionalKey := manifest.ConnectionField{Key: apiKeyField, Type: "secret", Title: "API key"}
	maximum := 8.0
	agentParallelRequests := manifest.ConnectionField{Key: manifest.ConnectionParallelRequests, Type: "number", Title: "Parallel requests", Default: float64(1), Min: &minimum, Max: &maximum, Step: &step}
	serverType := func(pluginID string) connectionport.Type {
		return connectionport.Type{PluginID: pluginID, PluginName: pluginID, IsAvailable: true, Declaration: manifest.ConnectionType{
			ID: "server", Title: "Server", Operations: getBothOperations(), Fields: []manifest.ConnectionField{address, key},
		}}
	}
	types := &fixedTypes{list: []connectionport.Type{
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
	w.Connections = connection.NewConnections(w.repository, types, w.Keys, NoUses{}, func() (string, error) {
		w.next++
		return fmt.Sprintf("c%08x", w.next), nil
	})
	return w
}

// Add creates a connection of the type with the address and, when key is not empty, the key.
func (w *World) Add(t *testing.T, typeID, name, address, key string) connection.Entry {
	t.Helper()
	request := connection.CreateRequest{TypeID: typeID, Name: name, Fields: map[string]any{addressField: address}}
	if key != "" {
		request.Secrets = map[string]string{apiKeyField: key}
	}
	entry, err := w.Connections.Create.Execute(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	return entry
}

// AddServer creates a connection of the type server of the first plugin.
func (w *World) AddServer(t *testing.T, name, address, key string) connection.Entry {
	t.Helper()
	return w.Add(t, ServerTypeID, name, address, key)
}

// Edit changes the name of the connection, which adds 1 to its revision.
func (w *World) Edit(t *testing.T, entry connection.Entry, name string) connection.Entry {
	t.Helper()
	changed, err := w.Connections.Update.Execute(t.Context(), connection.UpdateRequest{
		ID: entry.ID, Revision: entry.Revision, Name: name, Fields: entry.Fields,
	})
	if err != nil {
		t.Fatal(err)
	}
	return changed
}

// ChangeRevision adds 1 to the revision of the connection with no other change.
func (w *World) ChangeRevision(t *testing.T, connectionID string) {
	t.Helper()
	err := w.repository.ChangeConnections(func(list []connectionport.Connection) ([]connectionport.Connection, error) {
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

// Row returns a row of the connection with the model.
func Row(entry connection.Entry, model string) connectionport.Row {
	return connectionport.Row{ConnectionID: entry.ID, Model: model}
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

// ListUses returns no use.
func (NoUses) ListUses(string) ([]connectionport.Use, error) { return nil, nil }

// RemoveRows removes nothing.
func (NoUses) RemoveRows(string) error { return nil }

// Refresh configures no plugin.
func (NoUses) Refresh(context.Context, string) error { return nil }

// AddLimited creates a connection of the type limited, which runs two calls at once unless the field
// parallelRequests says otherwise. A parallelRequests of 0 leaves the field unset.
func (w *World) AddLimited(t *testing.T, name, address string, parallelRequests int) connection.Entry {
	t.Helper()
	fields := map[string]any{addressField: address}
	if parallelRequests > 0 {
		fields[manifest.ConnectionParallelRequests] = float64(parallelRequests)
	}
	entry, err := w.Connections.Create.Execute(t.Context(), connection.CreateRequest{
		TypeID: LimitedTypeID, Name: name, Fields: fields, Secrets: map[string]string{apiKeyField: "sk-" + name},
	})
	if err != nil {
		t.Fatal(err)
	}
	return entry
}

const addressField = "baseUrl"

func getBothOperations() []string { return []string{"complete", "embed"} }

func getCompleteOnly() []string { return []string{"complete"} }

// AddAgent creates a connection of the type agent, which has no address and no key and runs one call at a
// time.
func (w *World) AddAgent(t *testing.T, name string) connection.Entry {
	t.Helper()
	entry, err := w.Connections.Create.Execute(t.Context(), connection.CreateRequest{
		TypeID: AgentTypeID, Name: name, Fields: map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return entry
}
