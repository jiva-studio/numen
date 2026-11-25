package hostendpoint

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"slices"

	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// windowEndpoint serves HostWindowService for the plugins that call the host.
type windowEndpoint struct {
	endpoint *Endpoint
}

var _ numenv1connect.HostWindowServiceHandler = (*windowEndpoint)(nil)

func (e *windowEndpoint) begin(header http.Header) (WindowControl, error) {
	if _, err := e.endpoint.Authenticate(header.Get("Authorization")); err != nil {
		return nil, err
	}
	return e.endpoint.callers.Lent.GetPluginWindow(), nil
}

func mapWindowError(ctx context.Context, err error) (v1.ErrorCode, error) {
	switch {
	case err == nil:
		return v1.ErrorCode_ERROR_CODE_UNSPECIFIED, nil
	case errors.Is(err, plugin.ErrNoWindow):
		return v1.ErrorCode_ERROR_CODE_NO_WINDOW, nil
	case errors.Is(err, fs.ErrNotExist):
		return v1.ErrorCode_ERROR_CODE_NOT_FOUND, nil
	case errors.Is(err, plugin.ErrTabNotShown):
		return 0, connect.NewError(connect.CodeDeadlineExceeded, err)
	case errors.Is(err, plugin.ErrInvalidFocus):
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
		return connect.NewResponse(&v1.FocusNoteResponse{Error: v1.ErrorCode_ERROR_CODE_NO_WINDOW}), nil
	}
	ref, err := window.FocusNote(ctx, req.Msg.GetPath())
	code, err := mapWindowError(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.FocusNoteResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.FocusNoteResponse{Path: ref.Path, Title: ref.Title, Id: ref.ID}), nil
}

func (e *windowEndpoint) FocusFile(
	ctx context.Context, req *connect.Request[v1.FocusFileRequest],
) (*connect.Response[v1.FocusFileResponse], error) {
	window, err := e.begin(req.Header())
	if err != nil {
		return nil, err
	}
	if window == nil {
		return connect.NewResponse(&v1.FocusFileResponse{Error: v1.ErrorCode_ERROR_CODE_NO_WINDOW}), nil
	}
	spans := req.Msg.GetSpans()
	ranges := make([]domain.ByteRange, 0, len(spans))
	for _, span := range spans {
		ranges = append(ranges, domain.ByteRange{Start: int(span.GetStart()), End: int(span.GetEnd())})
	}
	code, err := mapWindowError(ctx, window.FocusFile(ctx, req.Msg.GetPath(), ranges))
	return connect.NewResponse(&v1.FocusFileResponse{Error: code}), err
}

func (e *windowEndpoint) ReadOpenTabs(
	ctx context.Context, req *connect.Request[v1.HostWindowServiceReadOpenTabsRequest],
) (*connect.Response[v1.HostWindowServiceReadOpenTabsResponse], error) {
	window, err := e.begin(req.Header())
	if err != nil {
		return nil, err
	}
	if window == nil {
		return connect.NewResponse(&v1.HostWindowServiceReadOpenTabsResponse{Error: v1.ErrorCode_ERROR_CODE_NO_WINDOW}), nil
	}
	open, err := window.ReadOpenTabs(ctx)
	code, err := mapWindowError(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.HostWindowServiceReadOpenTabsResponse{Error: code}), err
	}
	tabs := make([]*v1.OpenTab, 0, len(open.Tabs))
	for _, tab := range open.Tabs {
		tabs = append(tabs, convertTabToWire(tab))
	}
	return connect.NewResponse(&v1.HostWindowServiceReadOpenTabsResponse{Tabs: tabs, ActiveId: open.ActiveID}), nil
}

func (e *windowEndpoint) ActivateTab(
	ctx context.Context, req *connect.Request[v1.ActivateTabRequest],
) (*connect.Response[v1.ActivateTabResponse], error) {
	window, err := e.begin(req.Header())
	if err != nil {
		return nil, err
	}
	if window == nil {
		return connect.NewResponse(&v1.ActivateTabResponse{Error: v1.ErrorCode_ERROR_CODE_NO_WINDOW}), nil
	}
	id := req.Msg.GetTabId()
	open, err := window.ActivateTab(ctx, id)
	code, err := mapWindowError(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.ActivateTabResponse{Error: code}), err
	}
	index := slices.IndexFunc(open.Tabs, func(tab windowdomain.Tab) bool { return tab.ID == id })
	if index < 0 {
		return connect.NewResponse(&v1.ActivateTabResponse{Error: v1.ErrorCode_ERROR_CODE_NOT_FOUND}), nil
	}
	return connect.NewResponse(&v1.ActivateTabResponse{
		Tab:      convertTabToWire(open.Tabs[index]),
		ActiveId: open.ActiveID,
	}), nil
}

func convertTabToWire(tab windowdomain.Tab) *v1.OpenTab {
	return &v1.OpenTab{Id: tab.ID, Kind: tab.Kind, Path: tab.Path, Title: tab.Title, Place: tab.Place}
}
