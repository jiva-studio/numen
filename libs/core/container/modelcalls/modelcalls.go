// Package modelcalls wires the calls plugins ask for on models and the lists the editor asks for.
package modelcalls

import (
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

// OpenCalls returns the calls on models that SetModelCalls gives the plugin host, or nil for an installation
// that has no plugin host. A call made before SetModelCalls finds no connection.
func (c Config) OpenCalls() connectionport.ModelCalls {
	if c.PluginHost == nil || c.Connections == nil {
		return nil
	}
	host, isHost := c.PluginHost.(connectionport.ModelCallsHost)
	if !isHost {
		return nil
	}
	return modelcall.NewForwardedCalls(host.GetModelCalls)
}
