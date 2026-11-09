package assembly

import (
	"context"
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin-catalog/catalog"

	agentport "github.com/jiva-studio/numen/libs/core/usecase/agent/agentcontract"
	"github.com/jiva-studio/numen/libs/core/usecase/connection"
	modelcallport "github.com/jiva-studio/numen/libs/core/usecase/modelcall/modelcallcontract"
	"github.com/jiva-studio/numen/libs/core/usecase/modellist"
	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/usecase/run"
	"github.com/jiva-studio/numen/libs/core/usecase/secret"
	"github.com/jiva-studio/numen/libs/core/usecase/settings"
	"github.com/jiva-studio/numen/libs/core/usecase/tabkind"
)

// PluginServices is where this window forwards the calls it receives for the services plugins serve.
type PluginServices = interface {
	Serves(string) bool
	ServeHTTP(http.ResponseWriter, *http.Request)
	ServeRoute(http.ResponseWriter, *http.Request)
	ListFrameOrigins() []string
	GetFrameOrigins(pluginID string) []string
}

// PluginCatalog is the plugins this window lists and finds among the sources of the installation.
type PluginCatalog = interface {
	List(ctx context.Context) (catalog.Listing, error)
	Get(ctx context.Context, id string) (catalog.Entry, error)
	FindByTabKind(ctx context.Context, kind string) (catalog.Entry, error)
}

type pluginSecretAssembly interface {
	OpenSecrets() *secret.Secrets
	// OpenConnections is the use cases on the connections, and nil for an installation with no plugin host.
	OpenConnections() *connection.Connections
	GetPluginCommands() plugincommand.Commands
	// OpenChooseTabKind is the choice of the kind of tab a note opens in.
	OpenChooseTabKind() *tabkind.Choose
	// GetPluginCatalog is the catalog of plugins, and nil for an installation with none.
	GetPluginCatalog() PluginCatalog
}

type pluginWorkAssembly interface {
	modelAssembly
	EmitEvent(event string)
	OpenRetryPlugin() *settings.RetryPlugin
	ReportPluginFailures(tasks *run.Tasks)
	SetPluginWork(tasks *run.Tasks, models *run.Lock)
}

// modelAssembly is what the window requires for the models of a connection and the calls plugins make on them.
type modelAssembly interface {
	// OpenModelLists is the models and reasoning levels of a connection, and nil for an installation with no plugin host.
	OpenModelLists() *modellist.Lists
	// OpenModelCalls is the calls on models the window carries out for plugin views, and nil for an
	// installation with no plugin host.
	OpenModelCalls() modelcallport.ModelCalls
	// SetModelCalls lets the plugins ask for the operations on models; notices tells the person of a
	// connection set aside. An installation with no plugin host does nothing.
	SetModelCalls(notices modelcallport.Notices)
}

// PluginAssembly is what this window requires to manage, access, and report on plugins.
type PluginAssembly interface {
	GetPluginDir(id string) (string, bool)
	GetPluginServices() PluginServices
	GetAgent() agentport.Agent
	pluginSecretAssembly
	pluginWorkAssembly
}
