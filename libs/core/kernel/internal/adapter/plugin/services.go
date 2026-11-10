package plugin

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Services holds what the host lends to the plugins that call it: the vault, the texts, the window,
// the search, the queue of work, the completer and the provisioner of dependencies, and the
// endpoint a plugin reaches them through. The endpoint opens when the first plugin launches after
// one of them is set. It asks the callers who made a call and what the host lends, and the tools for
// the tools the plugins offer.
type Services struct {
	t           *table
	hostAddress string
	callers     hostendpoint.Callers
	tools       func() hostendpoint.PluginTools

	mu           sync.RWMutex
	completer    hostendpoint.Completer
	vault        *hostendpoint.VaultAccess
	texts        hostendpoint.TextVersions
	window       hostendpoint.WindowControl
	vaultList    hostendpoint.VaultList
	search       *hostendpoint.SearchAccess
	work         hostendpoint.WorkQueue
	dependencies pluginport.DependencyRegistry
	endpoint     *hostendpoint.Endpoint
}

func newServices(t *table) *Services {
	return &Services{t: t, hostAddress: hostendpoint.Address}
}

// SetPluginVault sets the vault exposed to plugins. Nil means none.
func (s *Services) SetPluginVault(vault *hostendpoint.VaultAccess) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.vault = vault
}

// GetPluginVault returns the vault exposed to plugins, or nil.
func (s *Services) GetPluginVault() *hostendpoint.VaultAccess {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.vault
}

// SetPluginTexts is the text versions the plugins work through the host.
func (s *Services) SetPluginTexts(texts hostendpoint.TextVersions) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.texts = texts
}

// GetPluginTexts is the text versions the plugins work, or nil.
func (s *Services) GetPluginTexts() hostendpoint.TextVersions {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.texts
}

// SetPluginWindow sets the window exposed to plugins. Nil means none.
func (s *Services) SetPluginWindow(window hostendpoint.WindowControl) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.window = window
}

// GetPluginWindow returns the window exposed to plugins, or nil.
func (s *Services) GetPluginWindow() hostendpoint.WindowControl {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.window
}

// SetPluginVaultList sets the list of vaults exposed to plugins. Nil means none.
func (s *Services) SetPluginVaultList(list hostendpoint.VaultList) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.vaultList = list
}

// GetPluginVaultList returns the list of vaults exposed to plugins, or nil.
func (s *Services) GetPluginVaultList() hostendpoint.VaultList {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.vaultList
}

// SetCompleter sets what answers the completions plugins ask for. The host
// endpoint starts serving when the first plugin launches after it is set.
func (s *Services) SetCompleter(completer hostendpoint.Completer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.completer = completer
}

// GetCompleter returns what answers the completions plugins ask for, or nil.
func (s *Services) GetCompleter() hostendpoint.Completer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.completer
}

// SetPluginSearch is the search index and the search floor the plugins work through the host.
func (s *Services) SetPluginSearch(search *hostendpoint.SearchAccess) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.search = search
}

// GetPluginSearch is the search index the plugins work, or nil.
func (s *Services) GetPluginSearch() *hostendpoint.SearchAccess {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.search
}

// SetPluginWork sets the queue and the list of tasks the plugins use.
func (s *Services) SetPluginWork(work hostendpoint.WorkQueue) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.work = work
}

// GetPluginWork returns the queue and the list of tasks the plugins use, or nil.
func (s *Services) GetPluginWork() hostendpoint.WorkQueue {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.work
}

// SetDependencies sets what provisions the dependencies the plugins declare and
// what the plugins ask about over the host endpoint.
func (s *Services) SetDependencies(catalog pluginport.DependencyRegistry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dependencies = catalog
}

// GetDependencies returns what provisions the dependencies, or nil when nothing does.
func (s *Services) GetDependencies() hostendpoint.Dependencies {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.dependencies == nil {
		return nil
	}
	return s.dependencies
}

// ObserveDependencies sets the function that observes each change of the state of a dependency. A
// host that provisions none observes none.
func (s *Services) ObserveDependencies(observer func(manifest.DependencyRequirement, plugindomain.DependencyStatus)) {
	s.mu.RLock()
	catalog := s.dependencies
	s.mu.RUnlock()
	if catalog != nil {
		catalog.SetObserver(observer)
	}
}

// GetPluginTools is the tools the plugins with the tools capability offer on the full server, which
// a plugin serves to programs outside Numen.
func (s *Services) GetPluginTools() hostendpoint.PluginTools {
	return s.tools()
}

// GetEndpointCallers is what the host lends the endpoint plugins call it through.
func (s *Services) GetEndpointCallers() hostendpoint.Callers {
	return s.callers
}

func (s *Services) openHostEndpoint(ctx context.Context) (int, error) {
	s.t.mu.RLock()
	isClosed := s.t.isClosed
	s.t.mu.RUnlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.endpoint != nil {
		return s.endpoint.GetPort(), nil
	}
	if (s.completer == nil && s.vault == nil && s.dependencies == nil && s.work == nil && s.search == nil) || isClosed {
		return 0, nil
	}
	endpoint, err := hostendpoint.Open(ctx, s.hostAddress, s.callers)
	if err != nil {
		return 0, err
	}
	s.endpoint = endpoint
	return endpoint.GetPort(), nil
}

func (s *Services) closeHostEndpoint(ctx context.Context) error {
	s.mu.Lock()
	endpoint := s.endpoint
	s.endpoint = nil
	s.mu.Unlock()
	if endpoint == nil {
		return nil
	}
	return endpoint.Close(ctx)
}

func (s *Services) clearPluginWork(pluginID string) {
	if work := s.GetPluginWork(); work != nil {
		work.ClearPlugin(pluginID)
	}
}

func (s *Services) clearPluginTasks(pluginID string) {
	s.mu.RLock()
	endpoint := s.endpoint
	s.mu.RUnlock()
	if endpoint != nil {
		endpoint.ClearPluginTasks(pluginID)
	}
}
