package theme

import (
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// NewRoute is where the theme service answers: its path and the handler of its calls. Each call goes
// through counted.
func NewRoute(handler numenv1connect.ThemeServiceHandler, counted connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewThemeServiceHandler(handler, counted)
}
