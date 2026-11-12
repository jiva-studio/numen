// Package shell serves a window that is a frame for the plugins of its target and has no domain of
// its own.
//
// It serves the window's pages, the tasks it displays, the themes it applies, the plugins of its
// target with the files of their frontends, and the calls for the services they serve. It provides the
// plugins with what the installation has: every vault registered, the reading of one into the index, and
// the notification that something moved.
package shell

import (
	"net/http"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
)

// API is what the page may ask of the window itself.
type API struct {
	Ensure               *indexing.EnsureIndexed
	Runs                 indexing.Runs
	Themes               numenv1connect.ThemeServiceHandler
	Window               *wire.Window
	GetFrontendPluginIDs func() []string
	GetPluginDir         func(id string) (string, bool)
	PluginServices       PluginServices
	Vaults               *registry.List
	ResolveLinks         *query.ResolveVaultLinks

	subscribers changeSubscribers
}

// PluginServices is where the calls for the services plugins serve are forwarded.
type PluginServices interface {
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
	return &wire.Window{Name: name, Tasks: tasks}
}
