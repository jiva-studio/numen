package hostendpoint

import (
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
)

// Services holds what the host lends to the plugins that call it: the completer, the vault, the
// texts, the window, the list of vaults, the search and the queue of work. What is not set is
// reported as nil, and a call that needs it is refused.
type Services struct {
	mu        sync.RWMutex
	completer Completer
	access    *vault.Access
	texts     TextVersions
	window    WindowControl
	vaultList vault.List
	search    *SearchAccess
	work      WorkQueue
}

// NewServices creates an empty set of services.
func NewServices() *Services { return &Services{} }

// SetCompleter sets what answers the completions plugins ask for.
func (s *Services) SetCompleter(completer Completer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.completer = completer
}

// GetCompleter returns what answers the completions plugins ask for, or nil.
func (s *Services) GetCompleter() Completer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.completer
}

// SetPluginVault sets the vault exposed to plugins. Nil means none.
func (s *Services) SetPluginVault(access *vault.Access) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.access = access
}

// GetPluginVault returns the vault exposed to plugins, or nil.
func (s *Services) GetPluginVault() *vault.Access {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.access
}

func (s *Services) SetPluginTexts(texts TextVersions) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.texts = texts
}

// GetPluginTexts returns the text versions the plugins work, or nil.
func (s *Services) GetPluginTexts() TextVersions {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.texts
}

// SetPluginWindow sets the window exposed to plugins. Nil means none.
func (s *Services) SetPluginWindow(window WindowControl) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.window = window
}

// GetPluginWindow returns the window exposed to plugins, or nil.
func (s *Services) GetPluginWindow() WindowControl {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.window
}

// SetPluginVaultList sets the list of vaults exposed to plugins. Nil means none.
func (s *Services) SetPluginVaultList(list vault.List) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.vaultList = list
}

func (s *Services) GetPluginVaultList() vault.List {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.vaultList
}

// SetPluginSearch sets the search index and the search floor the plugins work through the host.
func (s *Services) SetPluginSearch(search *SearchAccess) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.search = search
}

// GetPluginSearch returns the search index the plugins work, or nil.
func (s *Services) GetPluginSearch() *SearchAccess {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.search
}

// SetPluginWork sets the queue and the list of tasks the plugins use.
func (s *Services) SetPluginWork(work WorkQueue) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.work = work
}

// GetPluginWork returns the queue and the list of tasks the plugins use, or nil.
func (s *Services) GetPluginWork() WorkQueue {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.work
}
