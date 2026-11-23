package settings

import (
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// NewRoute is the path SettingsService is served at and the handler that serves it, and the handler it answers through.
func NewRoute(handler *Handler, options ...connect.HandlerOption) (path string, served http.Handler, service any) {
	path, served = numenv1connect.NewSettingsServiceHandler(handler, options...)
	return path, served, handler
}

// NewPluginDataRoute is the path PluginDataService is served at and the handler that serves it, and the handler it answers through.
func NewPluginDataRoute(handler *PluginFolders, options ...connect.HandlerOption) (path string, served http.Handler, service any) {
	path, served = numenv1connect.NewPluginDataServiceHandler(handler, options...)
	return path, served, handler
}
