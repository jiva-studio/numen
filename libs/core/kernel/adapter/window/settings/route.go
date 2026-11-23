package settings

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

type PluginDataService interface {
	GetPluginData(context.Context, *connect.Request[v1.GetPluginDataRequest]) (*connect.Response[v1.GetPluginDataResponse], error)
	DeletePluginData(context.Context, *connect.Request[v1.DeletePluginDataRequest]) (*connect.Response[v1.DeletePluginDataResponse], error)
}

// NewUnavailablePluginData is the service of an installation that keeps no plugin folders: it answers every
// call as unimplemented.
func NewUnavailablePluginData() PluginDataService {
	return numenv1connect.UnimplementedPluginDataServiceHandler{}
}

// NewUnavailablePathChoice is the choice of a build with no native dialog: it answers as unimplemented.
func NewUnavailablePathChoice() PathChoice {
	return numenv1connect.UnimplementedSettingsServiceHandler{}
}

// NewRoute is the path SettingsService is served at and the handler that serves it.
func NewRoute(handler *Handler, options ...connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewSettingsServiceHandler(handler, options...)
}

// NewPluginDataRoute is the path PluginDataService is served at and the handler that serves it.
func NewPluginDataRoute(handler PluginDataService, options ...connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewPluginDataServiceHandler(handler, options...)
}
