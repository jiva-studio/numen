package hostendpoint

import (
	"sync/atomic"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
)

// Services holds what the host lends to the plugins that call it: the calls on models, the vault, the
// texts, the window, the list of vaults, the search and the queue of work. What is not set is
// reported as nil, and a call that needs it is refused. The set is replaced whole at each change, so a
// call reads one consistent set without a lock.
type Services struct {
	set atomic.Pointer[lentSet]
}

type lentSet struct {
	modelCalls ModelCalls
	access     *vault.Access
	texts      TextVersions
	window     WindowControl
	vaultList  vault.List
	search     *SearchAccess
	work       WorkQueue
}

// NewServices creates an empty set of services.
func NewServices() *Services { return &Services{} }

func (s *Services) get() *lentSet {
	if current := s.set.Load(); current != nil {
		return current
	}
	return &lentSet{}
}

func (s *Services) change(apply func(*lentSet)) {
	for {
		old := s.set.Load()
		var next lentSet
		if old != nil {
			next = *old
		}
		apply(&next)
		if s.set.CompareAndSwap(old, &next) {
			return
		}
	}
}

// SetModelCalls sets what carries out the calls on models plugins ask for.
func (s *Services) SetModelCalls(calls ModelCalls) {
	s.change(func(set *lentSet) { set.modelCalls = calls })
}

// GetModelCalls returns what carries out the calls on models plugins ask for, or nil.
func (s *Services) GetModelCalls() ModelCalls { return s.get().modelCalls }

// SetPluginVault sets the vault exposed to plugins. Nil means none.
func (s *Services) SetPluginVault(access *vault.Access) {
	s.change(func(set *lentSet) { set.access = access })
}

// GetPluginVault returns the vault exposed to plugins, or nil.
func (s *Services) GetPluginVault() *vault.Access { return s.get().access }

func (s *Services) SetPluginTexts(texts TextVersions) {
	s.change(func(set *lentSet) { set.texts = texts })
}

// GetPluginTexts returns the text versions the plugins work, or nil.
func (s *Services) GetPluginTexts() TextVersions { return s.get().texts }

// SetPluginWindow sets the window exposed to plugins. Nil means none.
func (s *Services) SetPluginWindow(window WindowControl) {
	s.change(func(set *lentSet) { set.window = window })
}

// GetPluginWindow returns the window exposed to plugins, or nil.
func (s *Services) GetPluginWindow() WindowControl { return s.get().window }

// SetPluginVaultList sets the list of vaults exposed to plugins. Nil means none.
func (s *Services) SetPluginVaultList(list vault.List) {
	s.change(func(set *lentSet) { set.vaultList = list })
}

func (s *Services) GetPluginVaultList() vault.List { return s.get().vaultList }

// SetPluginSearch sets the search index and the search floor the plugins work through the host.
func (s *Services) SetPluginSearch(search *SearchAccess) {
	s.change(func(set *lentSet) { set.search = search })
}

// GetPluginSearch returns the search index the plugins work, or nil.
func (s *Services) GetPluginSearch() *SearchAccess { return s.get().search }

// SetPluginWork sets the queue and the list of tasks the plugins use.
func (s *Services) SetPluginWork(work WorkQueue) {
	s.change(func(set *lentSet) { set.work = work })
}

// GetPluginWork returns the queue and the list of tasks the plugins use, or nil.
func (s *Services) GetPluginWork() WorkQueue { return s.get().work }
