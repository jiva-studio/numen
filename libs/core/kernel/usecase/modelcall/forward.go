package modelcall

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"

	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
)

// ForwardedCalls sends each call to the calls get returns when the call is made. Where get returns nil
// every call finds no connection.
type ForwardedCalls struct {
	get func() connectionport.ModelCalls
}

func NewForwardedCalls(get func() connectionport.ModelCalls) *ForwardedCalls {
	return &ForwardedCalls{get: get}
}

// Call forwards to the calls of the moment.
func (f *ForwardedCalls) Call(ctx context.Context, callerPluginID string, request modelcall.Request, send modelcall.Send) ([]byte, error) {
	calls := f.get()
	if calls == nil {
		return nil, connection.ErrNoConnection
	}
	return calls.Call(ctx, callerPluginID, request, send)
}

// ListRows forwards to the calls of the moment; where there are none the plugin has no row.
func (f *ForwardedCalls) ListRows(callerPluginID, operation string) ([]modelcall.Row, error) {
	calls := f.get()
	if calls == nil {
		return []modelcall.Row{}, nil
	}
	return calls.ListRows(callerPluginID, operation)
}

// ListRowOptions forwards to the calls of the moment; where there are none no connection is found.
func (f *ForwardedCalls) ListRowOptions(
	ctx context.Context, callerPluginID, operation, connectionID, field string, options map[string]string,
) ([]modelcall.Option, error) {
	calls := f.get()
	if calls == nil {
		return nil, connection.ErrNotFound
	}
	return calls.ListRowOptions(ctx, callerPluginID, operation, connectionID, field, options)
}
