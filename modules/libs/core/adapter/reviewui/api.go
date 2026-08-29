// Package reviewui serves the window a person runs their cards in.
//
// It is a second adapter beside webui and not a part of it: what it answers is
// a different service over a different page, and what it holds is a slice of
// the installation — the registry and the four scenarios a review is made of.
// No scan runs behind it, nothing is embedded, and no agent is reached.
package reviewui

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	history "github.com/jiva-studio/numen/modules/libs/core/review"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/review"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// ErrNoVault is a question about a vault this installation does not hold.
var ErrNoVault = errors.New("no vault of that identity")

// ErrNoRun is an answer named against a run this window did not open. A run is
// one sitting at one window, and an answer belongs to the sitting it was given
// in.
var ErrNoRun = errors.New("no run of that name is open")

// API is what the page may ask.
//
// The runs it holds are the sittings open in this window. A run is a file in a
// vault that only this window appends to, and it is let go of when the window
// closes: what was written stands, and the next sitting opens a file of its own.
type API struct {
	Registry port.VaultRegistry
	// What this window does, one scenario to a field: what a vault owes, what to
	// ask next, where the answers leave a card, and the run they are appended to.
	Owed      review.Owed
	Session   review.Session
	Schedules review.Schedules
	Log       review.Log
	// Themes are the stylesheets the window may be dressed in, and the sizes it
	// is drawn and set at. They belong to the installation and not to a vault.
	Themes numenv1connect.ThemeServiceHandler
	// Now is when this is happening.
	Now func() time.Time
	// Settling is how long a vault that moved is given to be indexed before it
	// is counted a second time. Zero is Settling.
	Settling time.Duration

	mu   sync.Mutex
	runs map[string]sitting

	// following is everyone waiting to hear that a vault moved.
	following following
}

// sitting is one run open in this window, and the vault it was opened on.
type sitting struct {
	vault string
	run   *review.Run
}

// Vault is the vault of an identity, as the registry holds it.
func (a *API) Vault(id string) (domain.Vault, error) {
	all, err := a.Registry.All()
	if err != nil {
		return domain.Vault{}, err
	}
	for _, v := range all {
		if v.ID == id {
			return v, nil
		}
	}
	return domain.Vault{}, fmt.Errorf("%w: %s", ErrNoVault, id)
}

// opened starts a run and remembers it under its own name.
func (a *API) opened(ctx context.Context, v domain.Vault) (*review.Run, error) {
	run, err := a.Log.Open(ctx, v, a.now())
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.runs == nil {
		a.runs = make(map[string]sitting)
	}
	a.runs[run.Name()] = sitting{vault: v.ID, run: run}
	return run, nil
}

// running is the run of a name, opened by this window on this vault.
//
// The vault is part of what is asked for, because an answer is written to the
// vault its run was opened on: a run named against another vault would put a
// person's answer in a history it does not belong to, and an answer written is
// not written again.
func (a *API) running(vault, name string) (*review.Run, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	held, is := a.runs[name]
	if !is || held.vault != vault {
		return nil, fmt.Errorf("%w: %s", ErrNoRun, name)
	}
	return held.run, nil
}

func (a *API) now() time.Time {
	if a.Now == nil {
		return time.Now()
	}
	return a.Now()
}

// stamp is how an instant reaches the page: the one shape everything here
// writes, so the page parses one.
func stamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(history.Stamp)
}
