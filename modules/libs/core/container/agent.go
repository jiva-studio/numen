package container

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

var (
	errAgentUnavailable = errors.New("agent plugin is unavailable or not installed")
)

// AgentPort provides an implementation of port.Agent via the plugin host if available.
func (c Config) AgentPort() port.Agent {
	if c.PluginHost != nil && !c.SafeMode {
		if _, ok := c.PluginHost.FindPluginByCapability("agent"); ok {
			return hostAgentPort{host: c.PluginHost}
		}
		if _, ok := c.PluginHost.FindPluginByCapability("ai.agent.provider"); ok {
			return hostAgentPort{host: c.PluginHost}
		}
	}
	return nil
}

// AgentPortFor provides an implementation of port.Agent via the plugin host for a specific plugin ID.
func (c Config) AgentPortFor(pluginID string) port.Agent {
	if c.PluginHost != nil && !c.SafeMode {
		if _, ok := c.PluginHost.GetPlugin(pluginID); ok {
			return hostAgentPort{host: c.PluginHost, pluginID: pluginID}
		}
	}
	return nil
}

type hostAgentPort struct {
	host     port.PluginManager
	pluginID string
}

func (h hostAgentPort) openBridge(ctx context.Context) (*plugin.AgentBridge, error) {
	if h.host == nil {
		return nil, errAgentUnavailable
	}
	if h.pluginID != "" {
		return plugin.OpenAgentBridgeForPlugin(ctx, h.host, h.pluginID)
	}
	return plugin.OpenAgentBridge(ctx, h.host)
}

func (h hostAgentPort) StartRun(ctx context.Context, task port.Task) (port.Run, error) {
	bridge, err := h.openBridge(ctx)
	if err != nil {
		return nil, err
	}
	defer bridge.Close()
	return bridge.StartRun(ctx, task)
}

func (h hostAgentPort) EndConversation(ctx context.Context, conversation string) error {
	bridge, err := h.openBridge(ctx)
	if err != nil {
		return err
	}
	defer bridge.Close()
	return bridge.EndConversation(ctx, conversation)
}
