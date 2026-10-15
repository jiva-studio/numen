package container

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/port"
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

type hostAgentPort struct {
	host *plugin.Host
}

func (h hostAgentPort) StartRun(ctx context.Context, task port.Task) (port.Run, error) {
	bridge, err := plugin.OpenAgentBridge(ctx, h.host)
	if err != nil {
		return nil, err
	}
	run, err := bridge.StartRun(ctx, task)
	if err != nil {
		_ = bridge.Close()
		return nil, err
	}
	return run, nil
}

func (h hostAgentPort) EndConversation(ctx context.Context, conversation string) error {
	bridge, err := plugin.OpenAgentBridge(ctx, h.host)
	if err != nil {
		return err
	}
	defer bridge.Close()
	return bridge.EndConversation(ctx, conversation)
}
