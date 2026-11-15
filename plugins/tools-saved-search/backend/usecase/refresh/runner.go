// Package refresh keeps the results of the saved searches of a vault up to date: it marks them stale when
// the index changes, runs the ones that are due one after another on the search of the core, and writes
// each result into the cache file of its note.
package refresh

import (
	"context"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/ports"
)

// Settings are the cooldown between two runs of one search, in minutes.
type Settings struct {
	RefreshMinutes int
}

// Change is what a watch of a note receives: the start of a run, and the write of its cache with the
// time of the last good run.
type Change struct {
	ComputedAt time.Time
	IsRunning  bool
}

// Vault is what a run reads and writes of one vault.
type Vault struct {
	Notes  ports.Notes
	Lookup ports.NoteLookup
	Search ports.Search
	Store  ports.Store
}

// Dependencies are the vaults the runner opens and the task list of the host.
type Dependencies struct {
	Open  func(id domain.VaultID) Vault
	Tasks ports.Tasks
}

// Runner runs the saved searches of the vaults of one process. The runs of one vault go one after
// another; the runs of two vaults go on together. A run whose note or query has problems, or that fails,
// keeps the last good results and records the time and the reason, stale, so it is tried again after the
// cooldown; a failure of the host returns its error. A run the core reports stale stays stale.
type Runner struct {
	open     func(id domain.VaultID) Vault
	tasks    ports.Tasks
	settings func() Settings
	clock    ports.Clock
	vaults   *vaultLocks
	records  *sync.Mutex
	watchers *watchers
	timers   *timers
}

// New returns a runner over the dependencies, with the settings at the time of each run.
func New(dependencies Dependencies, settings func() Settings, clock ports.Clock) *Runner {
	return &Runner{
		open: dependencies.Open, tasks: dependencies.Tasks, settings: settings, clock: clock, vaults: newVaultLocks(),
		records: &sync.Mutex{}, watchers: newWatchers(), timers: newTimers(),
	}
}

// Watch returns a channel that receives each start of a run of the note and each write of its cache, and
// the function that ends the watch and closes the channel.
func (r *Runner) Watch(vault domain.VaultID, notePath string) (<-chan Change, func()) {
	return r.watchers.add(watchKey{vault: vault, path: notePath})
}

// IsRunning reports whether a run of the note is on its way.
func (r *Runner) IsRunning(vault domain.VaultID, notePath string) bool {
	return r.watchers.isRunning(watchKey{vault: vault, path: notePath})
}

// ListVaults returns, in order, the vaults the runner has run the searches of.
func (r *Runner) ListVaults() []domain.VaultID {
	return r.vaults.list()
}

func (r *Runner) now() time.Time {
	return r.clock.Now().UTC().Truncate(time.Second)
}

// vaultLocks lets one caller at a time run the searches of a vault.
type vaultLocks struct {
	mu    sync.Mutex
	locks map[domain.VaultID]chan struct{}
}

func newVaultLocks() *vaultLocks {
	return &vaultLocks{mu: sync.Mutex{}, locks: map[domain.VaultID]chan struct{}{}}
}

func (l *vaultLocks) list() []domain.VaultID {
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Sorted(maps.Keys(l.locks))
}

func (l *vaultLocks) lock(ctx context.Context, vault domain.VaultID) (func(), error) {
	l.mu.Lock()
	lock, isKnown := l.locks[vault]
	if !isKnown {
		lock = make(chan struct{}, 1)
		l.locks[vault] = lock
	}
	l.mu.Unlock()
	select {
	case lock <- struct{}{}:
		return func() { <-lock }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
