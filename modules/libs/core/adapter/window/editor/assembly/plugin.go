package assembly

import (
	"net/http"

	"github.com/jiva-studio/numen/modules/libs/core/port/agentport"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/settings"
)

// PluginServices is where this window forwards the calls it receives for the services plugins serve.
type PluginServices = interface {
	Serves(string) bool
	ServeHTTP(http.ResponseWriter, *http.Request)
}

// PluginAssembly is what this window asks to manage, reach, and report plugins.
type PluginAssembly interface {
	// GetPluginDir returns the root directory of an installed plugin by ID, or false if not found.
	GetPluginDir(id string) (string, bool)

	// GetPluginServices is where this window forwards the calls it receives for the services plugins
	// serve.
	GetPluginServices() PluginServices

	// GetAgent returns the agent provider for running tasks.
	GetAgent() agentport.Agent

	// EmitEvent sends an event notification to listeners and registered plugins.
	EmitEvent(event string)

	// The use cases that store and remove the key of a secret a plugin declares.
	OpenWritePluginSecret() *settings.WriteSecret
	OpenDeletePluginSecret() *settings.DeleteSecret
	// OpenRetryPlugin starts a plugin the host suspends once more.
	OpenRetryPlugin() *settings.RetryPlugin
	// ReportPluginFailures puts a task in the list for each plugin the host suspends.
	ReportPluginFailures(tasks *run.Tasks)
}
