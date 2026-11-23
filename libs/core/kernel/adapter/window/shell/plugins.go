package shell

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// FrontendPlugins lists the plugins for the window and locates their frontend files.
type FrontendPlugins struct {
	numenv1connect.UnimplementedPluginServiceHandler
	getIDs       func() []string
	getPluginDir func(string) (string, bool)
}

func NewFrontendPlugins(getIDs func() []string, getPluginDir func(string) (string, bool)) FrontendPlugins {
	return FrontendPlugins{getIDs: getIDs, getPluginDir: getPluginDir}
}

func (l FrontendPlugins) ListPlugins(
	context.Context, *connect.Request[v1.ListPluginsRequest],
) (*connect.Response[v1.ListPluginsResponse], error) {
	out := &v1.ListPluginsResponse{}
	for _, id := range l.getIDs() {
		out.Plugins = append(out.Plugins, &v1.PluginInfo{Id: id})
	}
	return connect.NewResponse(out), nil
}
