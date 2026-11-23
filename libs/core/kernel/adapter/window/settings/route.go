package settings

import (
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// NewRoute is the path SettingsService is served at and the handler that serves it.
func NewRoute(handler *Handler, options ...connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewSettingsServiceHandler(handler, options...)
}

// NewPluginDataRoute is the path PluginDataService is served at and the handler that serves it.
func NewPluginDataRoute(handler numenv1connect.PluginDataServiceHandler, options ...connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewPluginDataServiceHandler(handler, options...)
}
