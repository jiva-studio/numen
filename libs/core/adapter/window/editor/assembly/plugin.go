package assembly

import (
	"context"
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin-catalog/catalog"

	agentport "github.com/jiva-studio/numen/libs/core/usecase/completion/agentcontract"
	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/usecase/run"
	"github.com/jiva-studio/numen/libs/core/usecase/settings"
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
	OpenWritePluginSecret() *settings.WriteSecret
	OpenDeletePluginSecret() *settings.DeleteSecret
	GetPluginCommands() plugincommand.Commands
	// GetPluginCatalog is the catalog of plugins, and nil for an installation with none.
	GetPluginCatalog() PluginCatalog
}

type pluginWorkAssembly interface {
	EmitEvent(event string)
	OpenRetryPlugin() *settings.RetryPlugin
	ReportPluginFailures(tasks *run.Tasks)
	SetPluginWork(tasks *run.Tasks, models *run.Lock)
}

// PluginAssembly is what this window requires to manage, access, and report on plugins.
type PluginAssembly interface {
	GetPluginDir(id string) (string, bool)
	GetPluginServices() PluginServices
	GetAgent() agentport.Agent
	pluginSecretAssembly
	pluginWorkAssembly
}
