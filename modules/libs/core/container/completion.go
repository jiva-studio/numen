package container

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/port"
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

// setCompletionModels gives each enabled plugin that answers completions the
// models it says Complete answers with, every plugin asked at once. A plugin
// that fails or does not answer in time lists none, and the failure is reported.
func (c Config) setCompletionModels(infos []port.PluginInfo) {
	if c.SafeMode {
		return
	}
	ctx, stop := context.WithTimeout(context.Background(), programsLimit)
	defer stop()
	failed := make([]error, len(infos))
	var group sync.WaitGroup
	for at, info := range infos {
		if !slices.Contains(info.Capabilities, plugin.CapabilityAgentProvider) {
			continue
		}
		if _, enabled := c.PluginHost.GetPlugin(info.ID); !enabled {
			continue
		}
		group.Go(func() {
			infos[at].Models, failed[at] = listCompletionModels(ctx, c.PluginHost, info.ID)
		})
	}
	group.Wait()
	if err := errors.Join(failed...); err != nil {
		c.handleError(fmt.Errorf("list completion models: %w", err))
	}
}

// listCompletionModels asks one agent plugin for the models it answers completions with.
func listCompletionModels(ctx context.Context, host port.PluginActivator, pluginID string) ([]port.AgentModel, error) {
	bridge, err := plugin.OpenCompletionBridge(ctx, host, pluginID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = bridge.Close() }()
	return bridge.ListModels(ctx)
}
