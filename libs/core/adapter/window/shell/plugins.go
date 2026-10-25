package shell

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// pluginList answers which plugins the page loads: the ones of the window's target, by id.
type pluginList struct {
	numenv1connect.UnimplementedPluginServiceHandler
	getIDs func() []string
}

func (l pluginList) GetPlugins(
	context.Context, *connect.Request[v1.GetPluginsRequest],
) (*connect.Response[v1.GetPluginsResponse], error) {
	out := &v1.GetPluginsResponse{}
	if l.getIDs == nil {
		return connect.NewResponse(out), nil
	}
	for _, id := range l.getIDs() {
		out.Plugins = append(out.Plugins, &v1.PluginInfo{Id: id})
	}
	return connect.NewResponse(out), nil
}
