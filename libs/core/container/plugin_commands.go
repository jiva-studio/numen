package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/commandbridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/manifestlist"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// commandManifests lists what the manifests of the enabled plugins declare.
type commandManifests interface {
	ListCommands() []pluginport.PluginCommandEntry
	ListSlots() []pluginport.PluginSlotEntry
	ListHandlers() []pluginport.PluginCommandEntry
	GetParts(pluginID string) pluginport.PluginParts
}

// commandRuns runs a command in the backend of its plugin.
type commandRuns interface {
	RunCommand(ctx context.Context, call pluginport.PluginCommandCall) (pluginport.PluginCommandResult, error)
}

// pluginCommands is the commands the manifests declare, listed from the manifests, and run in the
// backends of their plugins.
type pluginCommands struct {
	commandManifests
	commandRuns
}

func (c Config) openPluginCommands() pluginCommands {
	if c.PluginHost == nil {
		return pluginCommands{manifestlist.NoManifests{}, commandbridge.NoCommands{}}
	}
	return pluginCommands{manifestlist.Open(c.PluginHost), commandbridge.Open(c.PluginHost)}
}
