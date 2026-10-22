// Package flashcards serves the window a person runs their cards in.
//
// The cards themselves are the plugins': what a vault owes, what to ask next and where an answer is
// written are served by the plugin that declares those services, and this window forwards the calls
// to it. What stands here is the window: its pages, the agent a card is asked about through, the
// tasks it draws, the themes it is dressed from, and what it lends the plugin of the installation —
// the vaults it knows, the reading of one into the index, and the word that something moved.
package flashcards

import (
	"context"
	"net/http"
	"sync/atomic"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// API is what the page may ask of the window itself.
type API struct {
	numenv1connect.UnimplementedAgentServiceHandler
	// IsVaultIndexed is whether the index already carries a vault.
	IsVaultIndexed *note.IsVaultIndexed
	// Themes are the stylesheets the window may be dressed in, and the sizes it
	// is drawn and set at. They belong to the installation and not to a vault.
	Themes numenv1connect.ThemeServiceHandler
	// Window is this window itself: everything being done behind it, and
	// everyone drawing it for the moment it goes.
	Window *wire.Window
	// PluginServices is where the calls for the services plugins serve are forwarded.
	PluginServices PluginServices

	// Opened is told which vault a plugin works for the person, when a session of review begins on it.
	// The agent works the vault the session is on.
	Opened func(context.Context, domain.Vault)

	// Unreachable is why an agent cannot be reached, when one cannot.
	Unreachable wire.Reason

	// agent is the agent a question about a card goes to. A window without one
	// answers that it has none, and the rest of it works as it did.
	agent atomic.Pointer[port.Agent]

	// currentCard is what the last question was asked about, which the tools
	// answer with. It is held here, and the question carries none of it: a
	// tool's answer is data where a question is read as instruction.
	currentCard atomic.Pointer[CurrentCard]

	indexRuns indexRuns

	// subscribers is everyone waiting to hear that a vault moved.
	subscribers changeSubscribers
}

// PluginServices is where the calls for the services plugins serve are forwarded.
type PluginServices = interface {
	// ServeHTTP answers a call for a service a plugin serves.
	ServeHTTP(w http.ResponseWriter, r *http.Request)
	// Serves reports whether an enabled plugin serves the service the path names.
	Serves(urlPath string) bool
}

// CurrentCard is what the last question was asked about. A window that has asked nothing is looking
// at nothing as far as anyone here knows.
type CurrentCard struct {
	Deck string
	Card string
	Face string
}

// GetCurrentCard is that, for whatever answers about it.
func (a *API) GetCurrentCard() CurrentCard {
	if on := a.currentCard.Load(); on != nil {
		return *on
	}
	return CurrentCard{}
}

// GetAsked is what the plugins that offer an agent tools are told of what the person is looking at.
func (a *API) GetAsked() map[string]string {
	on := a.GetCurrentCard()
	return map[string]string{"focus": on.Deck, "mark": on.Card, "face": on.Face}
}

// GetAgent is the agent a question about a card goes to, and nothing where
// this window has none.
func (a *API) GetAgent() port.Agent {
	if taking := a.agent.Load(); taking != nil {
		return *taking
	}
	return nil
}

// SetAgent names who takes those questions from now on. Nothing leaves the
// window with no agent.
func (a *API) SetAgent(taking port.Agent) {
	if taking == nil {
		a.agent.Store(nil)
		return
	}
	a.agent.Store(&taking)
}

// NewWindow is this window as the schema answers about it, over the list a
// build keeps of what it is doing. It is what the API's Window is built from,
// which is the whole of how a window outside this module names itself.
func NewWindow(tasks *run.Tasks) *wire.Window {
	if tasks == nil {
		tasks = run.NewTasks()
	}
	return &wire.Window{Named: wire.Review, Tasking: tasks}
}

// reportTask puts one piece of work in the list of what is being done behind the
// window, for a window that keeps one.
func (a *API) reportTask(pos run.Task) { a.Window.AddTask(pos) }

// finishTask takes one piece of work out of that list.
func (a *API) finishTask(id string) { a.Window.RemoveTask(id) }
