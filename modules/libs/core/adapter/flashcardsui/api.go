// Package flashcardsui serves the window a person runs their cards in.
//
// It is a second adapter beside webui and not a part of it: what it answers is
// a different service over a different page, and what it holds is a slice of
// the installation — the registry and the four scenarios flashcards is made of.
// Nothing is embedded here. A vault the index does not carry is read into it,
// and what this window writes into a vault is levelled in the index by the paths
// it touched.
package flashcardsui

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/task"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
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
// The runs it holds are the sittings open in this window, one to a vault. A run
// is a file in a vault that only this window appends to, and it is let go of
// when another sitting opens on that vault or the window closes: what was
// written stands, and the next sitting opens a file of its own.
type API struct {
	Registry port.VaultRegistry
	// What this window does, one scenario to a field: what a vault owes, what to
	// ask next, where the answers leave a card, the run they are appended to,
	// what each day of them came to, and what the deck being sat to is joined
	// to.
	Owed      flashcards.Owed
	Session   flashcards.Session
	Schedules flashcards.Schedules
	Log       flashcards.Log
	Counted   flashcards.Counted
	Joined    flashcards.Around
	// Presets is which preset each deck of a vault is scheduled by. Curves is
	// what the one control of a preset comes to over the whole range of its
	// goal.
	Presets flashcards.Presets
	Curves  flashcards.Curves
	// Notes is what a vault calls the note a preset stands in. A build with none
	// names a preset by nothing.
	Notes port.NoteQueries
	// Themes are the stylesheets the window may be dressed in, and the sizes it
	// is drawn and set at. They belong to the installation and not to a vault.
	Themes numenv1connect.ThemeServiceHandler
	// Tasking is everything being done behind the window. A build holding none
	// says there is nothing.
	Tasking *task.Tasks
	// Day is where one day of review gives way to the next. A build holding none
	// counts the day from midnight.
	Day history.Day
	// Now is when this is happening.
	Now func() time.Time

	// Sat is called with the vault a sitting has just opened on. What answers
	// about a card works one vault, and it is told which when the sitting is.
	Sat func(context.Context, domain.Vault)

	// Unreachable is why an agent cannot be reached, when one cannot.
	Unreachable atomic.Value

	// taking is the agent a question about a card goes to. A window without one
	// answers that it has none, and the rest of it works as it did.
	taking atomic.Pointer[port.Agent]

	mu sync.Mutex
	// runs is the sitting open on each vault, by the vault's identity.
	runs map[string]*flashcards.Run

	// reads is how a vault is brought up to date in the index, and behind is the
	// life those readings run for. walked is the vaults read since the window
	// opened, underway the ones being read now, and unreadable why the last
	// reading of one failed.
	reads      Read
	behind     context.Context
	walked     map[string]bool
	underway   map[string]bool
	unreadable map[string]string

	// following is everyone waiting to hear that a vault moved.
	following following
}

// Answering is the agent a question about a card goes to, and nothing where
// this window has none.
func (a *API) Answering() port.Agent {
	if taking := a.taking.Load(); taking != nil {
		return *taking
	}
	return nil
}

// Answers is who takes those questions from now on. Nothing leaves the window
// with no agent.
func (a *API) Answers(taking port.Agent) {
	if taking == nil {
		a.taking.Store(nil)
		return
	}
	a.taking.Store(&taking)
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
func (a *API) opened(ctx context.Context, v domain.Vault) (*flashcards.Run, error) {
	run, err := a.Log.Open(ctx, v, a.now())
	if err != nil {
		return nil, err
	}
	a.remember(v, run)
	// The agent works the vault the person is sitting to, and it is told which
	// once, when the sitting opens.
	if a.Sat != nil {
		a.Sat(ctx, v)
	}
	return run, nil
}

func (a *API) remember(v domain.Vault, run *flashcards.Run) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.runs == nil {
		a.runs = make(map[string]*flashcards.Run)
	}
	// One sitting to a vault: opening another lets go of the one before it, and
	// the file that one wrote is never appended to again.
	a.runs[v.ID] = run
}

// running is the run this window has open on a vault, and only under the name
// that run writes.
//
// The vault is part of what is asked for, because an answer is written to the
// vault its run was opened on: a run named against another vault would put a
// person's answer in a history it does not belong to, and an answer written is
// not written again. The name is asked for as well, so a page holding the name
// of a sitting that is over cannot go on writing to it.
func (a *API) running(vault, name string) (*flashcards.Run, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	run, is := a.runs[vault]
	if !is || run.Name() != name {
		return nil, fmt.Errorf("%w: %s", ErrNoRun, name)
	}
	return run, nil
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
