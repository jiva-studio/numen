// Package windowstate mounts the window service of the editor window: the tasks it runs and the
// flush of its pages at quit.
package windowstate

import (
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
)

// NewRoute is where the window service answers: its path and the handler of its calls. Each call
// goes through counted.
func NewRoute(handler numenv1connect.WindowServiceHandler, counted connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewWindowServiceHandler(handler, counted)
}
