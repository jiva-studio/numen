package hostendpoint

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/logs"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
)

var errNoLogs = errors.New("the host keeps no log entries for plugins")

// logsEndpoint answers the calls of LogService a plugin backend makes about itself. The plugin is
// the one the token names; a source in the request is ignored.
type logsEndpoint struct {
	numenv1connect.UnimplementedLogServiceHandler

	endpoint *Endpoint
}

func (e *logsEndpoint) WriteLog(
	_ context.Context, req *connect.Request[v1.WriteLogRequest],
) (*connect.Response[v1.WriteLogResponse], error) {
	pluginID, err := e.endpoint.Authenticate(req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}
	var writer LogWriter
	if e.endpoint.callers.Lent != nil {
		writer = e.endpoint.callers.Lent.GetPluginLogs()
	}
	if writer == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errNoLogs)
	}
	entry := wire.ReadLogEntry(req.Msg)
	entry.Source = pluginID
	if err := writer.Execute(entry); err != nil {
		if errors.Is(err, logs.ErrEmptyMessage) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.WriteLogResponse{}), nil
}
