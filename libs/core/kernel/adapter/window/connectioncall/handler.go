// Package connectioncall serves a window's ConnectionCallService: the operations a plugin view asks for,
// through the rows of the plugin the request names.
package connectioncall

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	shared "github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/connectioncall"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchcancel"
)

var errPluginToken = errors.New("a window serves ConnectionCallService to no plugin process")

// Calls carries out the operations plugins ask for.
type Calls = shared.Calls

// Rows lists the rows of the plugins and the values of their row fields.
type Rows = shared.Rows

// NewHandler is the handler over the calls and the rows of this installation. cancels holds the cancels of
// the calls it serves.
func NewHandler(calls Calls, rows Rows, cancels *searchcancel.Searches) *shared.Handler {
	return shared.NewHandler(func() Calls { return calls }, func() Rows { return rows }, identify, cancels)
}

func identify(ctx context.Context, header http.Header, requested string) (context.Context, string, error) {
	// A plugin process sends the token the host issued it, and the host endpoint serves it.
	if header.Get("Authorization") != "" {
		return ctx, "", connect.NewError(connect.CodePermissionDenied, errPluginToken)
	}
	return plugin.WithAskingPlugin(ctx, requested), requested, nil
}
