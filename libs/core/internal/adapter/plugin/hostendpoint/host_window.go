package hostendpoint

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// windowEndpoint answers HostWindowService for the plugins that call the host.
type windowEndpoint struct {
	endpoint *Endpoint
}

var _ numenv1connect.HostWindowServiceHandler = (*windowEndpoint)(nil)

func (e *windowEndpoint) FocusNote(
	_ context.Context, _ *connect.Request[v1.FocusNoteRequest],
) (*connect.Response[v1.FocusNoteResponse], error) {
	return connect.NewResponse(&v1.FocusNoteResponse{}), nil
}

func (e *windowEndpoint) FocusPassages(
	_ context.Context, _ *connect.Request[v1.FocusPassagesRequest],
) (*connect.Response[v1.FocusPassagesResponse], error) {
	return connect.NewResponse(&v1.FocusPassagesResponse{}), nil
}

func (e *windowEndpoint) ReadOpenTabs(
	_ context.Context, _ *connect.Request[v1.HostReadOpenTabsRequest],
) (*connect.Response[v1.HostReadOpenTabsResponse], error) {
	return connect.NewResponse(&v1.HostReadOpenTabsResponse{}), nil
}
