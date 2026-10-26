// Package shell serves a window that is a frame for the plugins of its target and holds no domain of
// its own.
//
// It serves the window's pages, the tasks it draws, the themes it is dressed in, the plugins of its
// target with the files of their frontends, and the calls for the services they serve. It lends the
// plugins what the installation has: every vault it knows, the reading of one into the index, and the
// word that something moved.
package shell

import (
	"net/http"

	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/usecase/agent"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/core/usecase/run"
	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// API is what the page may ask of the window itself.
type API struct {
	// IsVaultIndexed is whether the index already carries a vault.
	IsVaultIndexed *note.IsVaultIndexed
	// Themes are the stylesheets the window may be dressed in. They belong to the installation and
	// not to a vault.
	Themes numenv1connect.ThemeServiceHandler
	// Window is this window itself: everything being done behind it, and everyone drawing it for
	// the moment it goes.
	Window *wire.Window
	// GetPluginIDs are the plugins of the window's target.
	GetPluginIDs func() []string
	// GetPluginDir is the folder of an installed plugin, and false for one not installed.
	GetPluginDir func(id string) (string, bool)
	// PluginServices is where the calls for the services plugins serve are forwarded.
	PluginServices PluginServices
	// GetAgent is the agent the plugins' questions go to. Nil, or one returning nil, is a window with
	// none.
	GetAgent func() agent.Agent
	// ListVaults is every vault the installation holds, which the page offers to choose from. Nil is a
	// window that lists none.
	ListVaults func() ([]domain.Vault, error)

	indexRuns indexRuns

	// subscribers is everyone waiting to hear that a vault moved.
	subscribers changeSubscribers
}

// PluginServices is where the calls for the services plugins serve are forwarded.
type PluginServices = interface {
	http.Handler
	// Serves reports whether an enabled plugin serves the service the path names.
	Serves(urlPath string) bool
}

// NewWindow is the window named, over the list a build keeps of what it is doing. A nil list is
// replaced by an empty one.
func NewWindow(name string, tasks *run.Tasks) *wire.Window {
	if tasks == nil {
		tasks = run.NewTasks()
	}
	return &wire.Window{Named: name, Tasking: tasks}
}

// reportTask puts one piece of work in the list of what is being done behind the window.
func (a *API) reportTask(task run.Task) { a.Window.AddTask(task) }

// finishTask takes one piece of work out of that list.
func (a *API) finishTask(id string) { a.Window.RemoveTask(id) }
