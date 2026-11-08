// Package modelcalls wires the calls plugins ask for on models and the lists the editor asks for.
package modelcalls

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/providerbridge"
	"github.com/jiva-studio/numen/libs/core/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/usecase/connection"
	"github.com/jiva-studio/numen/libs/core/usecase/modelcall"
	"github.com/jiva-studio/numen/libs/core/usecase/modellist"
)

// Config wires the calls over the plugin host, the connections and the clock of an installation.
type Config struct {
	PluginHost  pluginport.PluginManager
	Connections *connection.Connections
	Clock       machineport.Clock
	Wait        machineport.Wait
	Rows        connectionport.Rows
}

// OpenLists returns the models and reasoning levels a provider gives, or nil for an installation that
// has no plugin host and so no connection.
func (c Config) OpenLists() *modellist.Lists {
	if c.PluginHost == nil || c.Connections == nil {
		return nil
	}
	return modellist.NewLists(c.Connections, providerbridge.NewSource(c.PluginHost))
}

// SetModelCalls lets the plugins ask the host for the operations on models. Nothing is set for an
// installation that has no plugin host, or a host that takes no calls.
func (c Config) SetModelCalls(notices connectionport.Notices) {
	if c.PluginHost == nil || c.Connections == nil {
		return
	}
	host, isHost := c.PluginHost.(connectionport.ModelCallsHost)
	if !isHost {
		return
	}
	host.SetModelCalls(modelcall.NewCalls(c.Connections, c.Rows, providerbridge.NewSource(c.PluginHost), c.Clock, c.Wait, notices))
}

// OpenCalls returns the calls on models that SetModelCalls gave the plugin host, or nil for an installation
// that has no plugin host. A call made before SetModelCalls finds no connection.
func (c Config) OpenCalls() connectionport.ModelCalls {
	if c.PluginHost == nil || c.Connections == nil {
		return nil
	}
	host, isHost := c.PluginHost.(connectionport.ModelCallsHost)
	if !isHost {
		return nil
	}
	return hostCalls{host: host}
}

// hostCalls forwards each call to the calls the plugin host holds when it is made, so the window and the
// plugins share one queue and one set of cooldowns.
type hostCalls struct{ host connectionport.ModelCallsHost }

func (h hostCalls) Complete(
	ctx context.Context, callerPluginID string, request connectionport.CompleteRequest, priority connectionport.Priority,
) (connectionport.CompleteResult, error) {
	calls := h.host.GetModelCalls()
	if calls == nil {
		return connectionport.CompleteResult{}, connectionport.ErrNoConnection
	}
	return calls.Complete(ctx, callerPluginID, request, priority)
}

func (h hostCalls) Embed(
	ctx context.Context, callerPluginID string, texts []string, priority connectionport.Priority,
) ([][]float32, error) {
	calls := h.host.GetModelCalls()
	if calls == nil {
		return nil, connectionport.ErrNoConnection
	}
	return calls.Embed(ctx, callerPluginID, texts, priority)
}

func (h hostCalls) GetEmbeddingProfile(ctx context.Context, callerPluginID string) (connectionport.EmbeddingProfile, error) {
	calls := h.host.GetModelCalls()
	if calls == nil {
		return connectionport.EmbeddingProfile{}, connectionport.ErrNoConnection
	}
	return calls.GetEmbeddingProfile(ctx, callerPluginID)
}
