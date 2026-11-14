package hostendpoint

import (
	"sync/atomic"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
)

// Lent is what the host lends to the plugins that call it: the calls on models, the rows, the vault, the
// texts, the window, the list of vaults, the search and the queue of work. What is not lent is nil, and a call
// that needs it is refused.
type Lent struct {
	ModelCalls ModelCalls
	RowLists   RowLists
	Vault      *vault.Access
	Texts      TextVersions
	Window     WindowControl
	VaultList  vault.List
	Search     *SearchAccess
	Work       WorkQueue
}

// Lender keeps the Lent the services read. It is replaced whole at each change, so a call reads one
// consistent set without a lock.
type Lender struct {
	current atomic.Pointer[Lent]
}

func NewLender() *Lender { return &Lender{} }

// Get returns the set lent now.
func (l *Lender) Get() Lent {
	if current := l.current.Load(); current != nil {
		return *current
	}
	return Lent{}
}

// Update changes the set lent: apply receives a copy of it to change.
func (l *Lender) Update(apply func(*Lent)) {
	for {
		old := l.current.Load()
		var next Lent
		if old != nil {
			next = *old
		}
		apply(&next)
		if l.current.CompareAndSwap(old, &next) {
			return
		}
	}
}

// Services gives the plugins' calls what the host lends, as it is when the call is made.
type Services struct {
	source func() Lent
}

// NewServices creates services that read what is lent from source at each call.
func NewServices(source func() Lent) *Services { return &Services{source: source} }

// GetModelCalls returns what carries out the calls on models plugins ask for, or nil.
func (s *Services) GetModelCalls() ModelCalls { return s.source().ModelCalls }

// GetRowLists returns what lists the rows of the plugins and the values of their row fields, or nil.
func (s *Services) GetRowLists() RowLists { return s.source().RowLists }

// GetPluginVault returns the vault exposed to plugins, or nil.
func (s *Services) GetPluginVault() *vault.Access { return s.source().Vault }

// GetPluginTexts returns the text versions the plugins work, or nil.
func (s *Services) GetPluginTexts() TextVersions { return s.source().Texts }

// GetPluginWindow returns the window exposed to plugins, or nil.
func (s *Services) GetPluginWindow() WindowControl { return s.source().Window }

func (s *Services) GetPluginVaultList() vault.List { return s.source().VaultList }

// GetPluginSearch returns the search index the plugins work, or nil.
func (s *Services) GetPluginSearch() *SearchAccess { return s.source().Search }

// GetPluginWork returns the queue and the list of tasks the plugins use, or nil.
func (s *Services) GetPluginWork() WorkQueue { return s.source().Work }
