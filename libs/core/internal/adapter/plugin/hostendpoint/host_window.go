package hostendpoint

import (
	"context"
	"errors"
	"io/fs"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/notes/domain"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

// windowEndpoint answers HostWindowService for the plugins that call the host.
type windowEndpoint struct {
	endpoint *Endpoint
}

var _ numenv1connect.HostWindowServiceHandler = (*windowEndpoint)(nil)

// begin authorizes a window call and returns the window, nil where the host has none.
func (e *windowEndpoint) begin(header http.Header) (pluginport.PluginWindow, error) {
	if _, err := e.endpoint.authorize(header.Get("Authorization"), manifest.PermissionWindowFocus); err != nil {
		return nil, err
	}
	return e.endpoint.callers.GetPluginWindow(), nil
}

// answerWindow sorts the failure of a window call: an answer the protocol names is a code, an
// argument the use case refuses is invalid, the end of the caller's context is cancelled, and
// anything else is internal.
func answerWindow(ctx context.Context, err error) (v1.HostWindowErrorCode, error) {
	switch {
	case err == nil:
		return v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_UNSPECIFIED, nil
	case errors.Is(err, pluginport.ErrNoWindow):
		return v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_NO_WINDOW, nil
	case errors.Is(err, fs.ErrNotExist):
		return v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_NOT_FOUND, nil
	case errors.Is(err, pluginport.ErrInvalidFocus):
		return 0, connect.NewError(connect.CodeInvalidArgument, err)
	case ctx.Err() != nil:
		return 0, connect.NewError(connect.CodeCanceled, ctx.Err())
	}
	return 0, connect.NewError(connect.CodeInternal, err)
}

func (e *windowEndpoint) FocusNote(
	ctx context.Context, req *connect.Request[v1.FocusNoteRequest],
) (*connect.Response[v1.FocusNoteResponse], error) {
	window, err := e.begin(req.Header())
	if err != nil {
		return nil, err
	}
	if window == nil {
		return connect.NewResponse(&v1.FocusNoteResponse{Error: v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_NO_WINDOW}), nil
	}
	ref, err := window.FocusNote(ctx, req.Msg.GetPath())
	code, err := answerWindow(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.FocusNoteResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.FocusNoteResponse{Path: ref.Path, Title: ref.Title, Id: ref.ID}), nil
}

func (e *windowEndpoint) FocusPassages(
	ctx context.Context, req *connect.Request[v1.FocusPassagesRequest],
) (*connect.Response[v1.FocusPassagesResponse], error) {
	window, err := e.begin(req.Header())
	if err != nil {
		return nil, err
	}
	if window == nil {
		return connect.NewResponse(&v1.FocusPassagesResponse{Error: v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_NO_WINDOW}), nil
	}
	spans := req.Msg.GetPassages()
	passages := make([]domain.ByteRange, 0, len(spans))
	for _, span := range spans {
		start := int(span.GetStart())
		passages = append(passages, domain.ByteRange{Start: start, End: start + int(span.GetLength())})
	}
	summary, err := window.FocusPassages(ctx, req.Msg.GetPath(), passages)
	code, err := answerWindow(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.FocusPassagesResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.FocusPassagesResponse{Summary: summary}), nil
}

func (e *windowEndpoint) ReadOpenTabs(
	ctx context.Context, req *connect.Request[v1.HostWindowServiceReadOpenTabsRequest],
) (*connect.Response[v1.HostWindowServiceReadOpenTabsResponse], error) {
	window, err := e.begin(req.Header())
	if err != nil {
		return nil, err
	}
	if window == nil {
		return connect.NewResponse(&v1.HostWindowServiceReadOpenTabsResponse{Error: v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_NO_WINDOW}), nil
	}
	open, err := window.ReadOpenTabs(ctx)
	code, err := answerWindow(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.HostWindowServiceReadOpenTabsResponse{Error: code}), err
	}
	tabs := make([]*v1.Tab, 0, len(open.Tabs))
	for _, tab := range open.Tabs {
		tabs = append(tabs, &v1.Tab{Id: tab.ID, Kind: tab.Kind, Path: tab.Path, Title: tab.Title, State: tab.State})
	}
	return connect.NewResponse(&v1.HostWindowServiceReadOpenTabsResponse{Tabs: tabs, ActiveId: open.ActiveID}), nil
}
