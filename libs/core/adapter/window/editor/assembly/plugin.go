package assembly

import (
	"net/http"

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

type pluginSecretAssembly interface {
	OpenWritePluginSecret() *settings.WriteSecret
	OpenDeletePluginSecret() *settings.DeleteSecret
	GetPluginCommands() plugincommand.Commands
}

type pluginWorkAssembly interface {
	EmitEvent(event string)
	OpenRetryPlugin() *settings.RetryPlugin
	ReportPluginFailures(tasks *run.Tasks)
	SetPluginWork(tasks *run.Tasks, models *run.Lock)
}

// PluginAssembly is what this window asks to manage, reach, and report plugins.
type PluginAssembly interface {
	GetPluginDir(id string) (string, bool)
	GetPluginServices() PluginServices
	GetAgent() agentport.Agent
	pluginSecretAssembly
	pluginWorkAssembly
}
