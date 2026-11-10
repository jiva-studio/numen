package plugin

import (
	"errors"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/eventbus"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

const settingEnabled = "enabled"

// table is the state of the plugins that every role of the host reads and changes under one lock:
// which plugins are known, the supervisor and the endpoint of each, and the stored settings that
// decide whether a plugin runs. A method named Locked expects the caller to hold mu. A scan of the
// folders holds discovering for writing and an activation holds it for reading, so that a plugin is
// not started while the set of plugins changes.
type table struct {
	mu            sync.RWMutex
	discovering   sync.RWMutex
	plugins       map[string]manifest.Discovered
	onDisk        map[string]bool
	builtIns      map[string]builtInPlugin
	problems      []manifestjson.DiscoveryProblem
	supervisors   map[string]*lifecycle.Supervisor
	endpoints     map[string]plugindomain.Endpoint
	clients       *transport.Clients
	listenerPorts map[string]map[string]int
	settings      map[string]map[string]any
	activation    lifecycle.ActivationPolicy
	safeMode      bool
	isClosed      bool
	vaultPath     string
}

func newTable() *table {
	return &table{
		plugins:       make(map[string]manifest.Discovered),
		onDisk:        make(map[string]bool),
		builtIns:      make(map[string]builtInPlugin),
		supervisors:   make(map[string]*lifecycle.Supervisor),
		endpoints:     make(map[string]plugindomain.Endpoint),
		clients:       transport.NewClients(),
		listenerPorts: make(map[string]map[string]int),
		settings:      make(map[string]map[string]any),
		activation:    lifecycle.DefaultActivationPolicy(),
	}
}

func (t *table) getPluginsInOrderLocked() []manifest.Discovered {
	result := make([]manifest.Discovered, 0, len(t.plugins))
	for _, p := range t.plugins {
		result = append(result, p)
	}
	slices.SortFunc(result, func(a, b manifest.Discovered) int { return strings.Compare(a.Manifest.ID, b.Manifest.ID) })
	return result
}

func (t *table) isPluginEnabledLocked(pluginID string) bool {
	settings := maps.Clone(t.settings[pluginID])
	if p, ok := t.plugins[pluginID]; ok {
		settings = p.Manifest.ResolveSettings(settings)
	}
	if val, ok := settings[settingEnabled].(bool); ok {
		return val
	}
	return true
}

func (t *table) isPluginActiveLocked(pluginID string) bool {
	if sup, ok := t.supervisors[pluginID]; ok {
		return sup.GetState() == lifecycle.StateActive || sup.GetState() == lifecycle.StateIdle
	}
	return false
}

func (t *table) getPluginSettingsLocked(pluginID string) map[string]any {
	rawStored := t.settings[pluginID]
	result := maps.Clone(rawStored)
	if p, ok := t.plugins[pluginID]; ok {
		result = p.Manifest.ResolveSettings(result)
	}
	if result == nil {
		result = map[string]any{}
	}
	if p, ok := t.plugins[pluginID]; ok {
		for _, name := range p.Manifest.GetSecretNames() {
			delete(result, name)
		}
	}
	return result
}

func (t *table) getRequirementIDsLocked(pluginID string) []string {
	p, ok := t.plugins[pluginID]
	if !ok {
		return nil
	}
	var ids []string
	for _, req := range p.Manifest.GetDependencyRequirements(p.Dir, t.getPluginSettingsLocked(pluginID)) {
		ids = append(ids, req.ID)
	}
	return ids
}

func (t *table) forgetEndpointLocked(pluginID string) {
	delete(t.endpoints, pluginID)
	delete(t.listenerPorts, pluginID)
}

func (t *table) getBuiltInLocked(pluginID string) (builtInPlugin, bool) {
	builtIn, ok := t.builtIns[pluginID]
	return builtIn, ok
}

// builtInPlugin is a plugin linked into the host's binary: the plugin its manifest describes and
// what makes the handler that serves its backend from the variables a launched process of it is
// given.
type builtInPlugin struct {
	plugin manifest.Discovered
	serve  func(vars map[string]string) (http.Handler, error)
}

var (
	errPluginNotFound = errors.New("plugin not found")
	// ErrSafeModeActive is what a call fails with while the host runs in safe mode.
	ErrSafeModeActive = errors.New("plugin host is running in safe mode")
	errHostClosed     = errors.New("plugin host is shut down")
)

// errorSink receives the problems the host meets outside a call, such as a failed shutdown.
type errorSink struct {
	mu      sync.RWMutex
	handler eventbus.ErrorHandler
}

func (s *errorSink) set(handler eventbus.ErrorHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handler = handler
}

func (s *errorSink) report(err error) {
	s.mu.RLock()
	handler := s.handler
	s.mu.RUnlock()
	if handler != nil {
		handler(err)
	}
}
