package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/completion"
)

// newCompletionRouter answers the questions plugins ask of a model with the
// agent each plugin's setting names, else the default under ai.default in the
// settings file, both read at every question, and the agent plugins of the host.
func (c Config) newCompletionRouter() (*completion.Router, error) {
	return completion.NewRouter(
		func() completion.Config { return c.getSettingsOrDefaults().AI },
		c.PluginHost,
		plugin.NewCompletionAgents(c.PluginHost),
	)
}

// setCompleter has the plugin host answer the questions its plugins ask of a
// model. Safe mode serves none.
func (c Config) setCompleter() {
	if c.PluginHost == nil || c.SafeMode {
		return
	}
	router, err := c.newCompletionRouter()
	if err != nil {
		c.handleError(err)
		return
	}
	c.PluginHost.SetCompleter(router)
}
