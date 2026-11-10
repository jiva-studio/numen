// Package sessions keeps the run open on each vault.
package sessions

import (
	"errors"
	"fmt"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/flashcards"
)

// ErrNoRun is an answer named against a run this backend did not open. A run is one session at one
// window, and an answer belongs to the session it was given in.
var ErrNoRun = errors.New("no run of that name is open")

// Registry is the run open on each vault, by the vault's identity. One session to a vault: opening
// another releases the one before it, and the file that one wrote is never appended to again.
type Registry struct {
	mu  sync.Mutex
	run map[domain.VaultID]*flashcards.LogWriter
}

// NewRegistry returns a registry in which no run is open.
func NewRegistry() *Registry {
	return &Registry{mu: sync.Mutex{}, run: make(map[domain.VaultID]*flashcards.LogWriter)}
}

// Store makes run the one open on the vault v.
func (r *Registry) Store(v domain.Vault, run *flashcards.LogWriter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.run[v.ID] = run
}

// GetRun returns the run named, if it is the one open on the vault.
func (r *Registry) GetRun(vault domain.VaultID, name string) (*flashcards.LogWriter, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	run, is := r.run[vault]
	if !is || run.GetName() != name {
		return nil, fmt.Errorf("%w: %s", ErrNoRun, name)
	}
	return run, nil
}
