package hostendpoint

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/connectioncall"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchcancel"
)

// newConnectionCalls answers ConnectionCallService for the plugins that call the host. The connections are
// the rows of the calling plugin for the operation, so a plugin reaches only the operations its settings of
// rows name or, with connections:use:<operation>, the connections that serve that operation.
func newConnectionCalls(e *Endpoint, cancels *searchcancel.Searches) numenv1connect.ConnectionCallServiceHandler {
	return connectioncall.NewHandler(
		func() connectioncall.Calls { return e.callers.Lent.GetModelCalls() },
		func() connectioncall.Rows { return e.callers.Lent.GetRowLists() },
		e.identifyCaller,
		cancels,
	)
}

// identifyCaller is the plugin the token names. A request that names another plugin is refused: a plugin
// reaches the rows of itself only.
func (e *Endpoint) identifyCaller(ctx context.Context, header http.Header, requested string) (context.Context, string, error) {
	pluginID, err := e.Authenticate(header.Get("Authorization"))
	if err != nil {
		return ctx, "", err
	}
	if requested != "" && requested != pluginID {
		return ctx, "", connect.NewError(connect.CodePermissionDenied, fmt.Errorf("plugin %s asks for the rows of plugin %s", pluginID, requested))
	}
	return ctx, pluginID, nil
}
