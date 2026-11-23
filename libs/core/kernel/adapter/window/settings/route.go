package settings

import (
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// NewRoute is where SettingsService is mounted, what the schema generated to answer it, and the
// handler it answers through.
func NewRoute(handler *Handler, options ...connect.HandlerOption) (path string, served http.Handler, service any) {
	path, served = numenv1connect.NewSettingsServiceHandler(handler, options...)
	return path, served, handler
}

// NewPluginDataRoute is where PluginDataService is mounted, what the schema generated to answer it, and the
// handler it answers through.
func NewPluginDataRoute(handler *PluginFolders, options ...connect.HandlerOption) (path string, served http.Handler, service any) {
	path, served = numenv1connect.NewPluginDataServiceHandler(handler, options...)
	return path, served, handler
}
