package servertest

import (
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
)

// NewHandler returns the handler of a server with the services mount registers on it, which
// authenticates with the NUMEN_PLUGIN_TOKEN entry of vars. A server started with server.Run takes its
// token and its ready line over standard input and output and stops on signals, so a test reaches the
// services of a plugin through this handler.
func NewHandler(vars map[string]string, mount func(*server.Server) error) (http.Handler, error) {
	return server.NewLinkedHandler(vars, mount)
}
