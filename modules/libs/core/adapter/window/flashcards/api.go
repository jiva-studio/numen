// Package flashcards serves the window a person runs their cards in.
//
// What it answers is a different service over a different page, and what it
// holds is a slice of the installation — the registry and the four scenarios
// flashcards is made of.
// Nothing is embedded here. A vault the index does not carry is read into it,
// and what this window writes into a vault is levelled in the index by the paths
// it touched.
package flashcards

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
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
	CardsDue      flashcards.CountCardsDue
	Session       flashcards.Session
	Schedules     flashcards.Schedules
	Log           flashcards.Log
	Counted       flashcards.CountReviews
	Neighbourhood flashcards.ShowNeighbourhood
	// Presets is which preset each deck of a vault is scheduled by.
	Presets flashcards.Presets
	// Notes is what a vault calls the note a preset stands in. A build with none
	// names a preset by nothing.
	Notes port.NoteQueries
	// Themes are the stylesheets the window may be dressed in, and the sizes it
	// is drawn and set at. They belong to the installation and not to a vault.
	Themes numenv1connect.ThemeServiceHandler
	// Window is this window itself: everything being done behind it, and
	// everyone drawing it for the moment it goes.
	Window *wire.Window
	// Day is where one day of review gives way to the next. A build holding none
	// counts the day from midnight.
	Day review.Day
	// Now is when this is happening, and what every scenario this window runs
	// is handed. A window that names none reads this machine's clock, which an
	// adapter is allowed to and a scenario is not.
	Now port.Clock

	// Opened is called with the vault a sitting has just opened on. What answers
	// about a card works one vault, and it is told which when the sitting is.
	Opened func(context.Context, domain.Vault)

	// Unreachable is why an agent cannot be reached, when one cannot.
	Unreachable wire.Reason

	// agent is the agent a question about a card goes to. A window without one
	// answers that it has none, and the rest of it works as it did.
	agent atomic.Pointer[port.Agent]

	// showing is the card the last question was asked about, which the tools
	// answer with. It is held rather than written into the question: a deck is
	// named by whoever synced it, and a name in a question is read as
	// instruction where a tool's answer is data.
	showing atomic.Pointer[CurrentCard]

	runs     sittings
	readings readings

	// listeners is everyone waiting to hear that a vault moved.
	listeners following
}

// sittings is the run open on each vault, by the vault's identity. One sitting
// to a vault: opening another lets go of the one before it, and the file that
// one wrote is never appended to again.
type sittings struct {
	mu  sync.Mutex
	run map[domain.VaultID]*flashcards.LogWriter
}

func (s *sittings) remember(v domain.Vault, run *flashcards.LogWriter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.run == nil {
		s.run = make(map[domain.VaultID]*flashcards.LogWriter)
	}
	s.run[v.ID] = run
}

// named is the run this window has open on a vault, and only under the name
// that run writes.
//
// The vault is part of what is asked for, because an answer is written to the
// vault its run was opened on: a run named against another vault would put a
// person's answer in a history it does not belong to, and an answer written is
// not written again. The name is asked for as well, so a page holding the name
// of a sitting that is over cannot go on writing to it.
func (s *sittings) named(vault domain.VaultID, name string) (*flashcards.LogWriter, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, is := s.run[vault]
	if !is || run.Name() != name {
		return nil, fmt.Errorf("%w: %s", ErrNoRun, name)
	}
	return run, nil
}

// CurrentCard is the card in front of the person, as the last question said
// it. A window that has asked nothing is looking at no card as far as anyone
// here knows.
type CurrentCard struct {
	Deck string
	Card string
	Face string
}

// Current is that card, for whatever answers about it.
func (a *API) Current() CurrentCard {
	if on := a.showing.Load(); on != nil {
		return *on
	}
	return CurrentCard{}
}

// Answering is the agent a question about a card goes to, and nothing where
// this window has none.
func (a *API) Answering() port.Agent {
	if taking := a.agent.Load(); taking != nil {
		return *taking
	}
	return nil
}

// Answers is who takes those questions from now on. Nothing leaves the window
// with no agent.
func (a *API) Answers(taking port.Agent) {
	if taking == nil {
		a.agent.Store(nil)
		return
	}
	a.agent.Store(&taking)
}

// Vault is the vault of an identity, as the registry holds it.
func (a *API) Vault(id string) (domain.Vault, error) {
	all, err := a.Registry.All()
	if err != nil {
		return domain.Vault{}, err
	}
	for _, v := range all {
		if string(v.ID) == id {
			return v, nil
		}
	}
	return domain.Vault{}, fmt.Errorf("%w: %s", ErrNoVault, id)
}

// opened starts a run and remembers it under its own name.
func (a *API) opened(ctx context.Context, v domain.Vault) (*flashcards.LogWriter, error) {
	run, err := a.Log.Open(ctx, v, a.now())
	if err != nil {
		return nil, err
	}
	a.runs.remember(v, run)
	// The agent works the vault the person is sitting to, and it is told which
	// once, when the sitting opens.
	if a.Opened != nil {
		a.Opened(ctx, v)
	}
	return run, nil
}

// Watching is this window as the schema answers about it, over the list a build
// keeps of what it is doing. It is what the API's Window is built from, which
// is the whole of how a window outside this module names itself.
func Watching(tasks *task.Tasks) *wire.Window {
	return &wire.Window{Named: wire.Review, Tasking: tasks}
}

// say puts one piece of work in the list of what is being done behind the
// window, for a window that keeps one.
func (a *API) say(at task.Task) { a.Window.Say(at) }

// finished takes one piece of work out of that list.
func (a *API) finished(id string) { a.Window.Finished(id) }

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
	return t.UTC().Format(review.Stamp)
}
