package editor

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/tablinks"
)

// ReadDetachedTab returns the tab of a pending detach, with the state its kind saved.
func (h *workspaceHandler) ReadDetachedTab(
	_ context.Context, r *connect.Request[v1.ReadDetachedTabRequest],
) (*connect.Response[v1.ReadDetachedTabResponse], error) {
	if h.state.TabLinks == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoTabLinks)
	}
	out := &v1.ReadDetachedTabResponse{}
	if tab, ok := h.state.TabLinks.ReadDetached(r.Msg.GetTabId()); ok {
		out.Tab = convertMovedTabToWire(tab)
	}
	return connect.NewResponse(out), nil
}

// MoveTab sends a tab to another window and responds after the target window
// accepts it, its watch closes, or the move times out.
func (h *workspaceHandler) MoveTab(
	ctx context.Context, r *connect.Request[v1.MoveTabRequest],
) (*connect.Response[v1.MoveTabResponse], error) {
	if h.state.TabLinks == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoTabLinks)
	}
	if r.Msg.GetWindowId() == "" || r.Msg.GetTargetWindowId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNoWindowID)
	}
	result := h.state.TabLinks.Move(r.Msg.GetWindowId(), r.Msg.GetTargetWindowId(), convertMovedTabFromWire(r.Msg.GetTab()))
	select {
	case <-ctx.Done():
		return nil, connect.NewError(connect.CodeCanceled, ctx.Err())
	case outcome := <-result:
		return connect.NewResponse(&v1.MoveTabResponse{Error: convertMoveErrorToWire(outcome)}), nil
	}
}

// AcceptTabMove completes the move of a tab into the window that opened it.
func (h *workspaceHandler) AcceptTabMove(
	_ context.Context, r *connect.Request[v1.AcceptTabMoveRequest],
) (*connect.Response[v1.AcceptTabMoveResponse], error) {
	if h.state.TabLinks == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoTabLinks)
	}
	outcome := h.state.TabLinks.Accept(r.Msg.GetWindowId(), r.Msg.GetTabId())
	return connect.NewResponse(&v1.AcceptTabMoveResponse{Error: convertMoveErrorToWire(outcome)}), nil
}

// HighlightWindow turns the outline of a window on or off.
func (h *workspaceHandler) HighlightWindow(
	_ context.Context, r *connect.Request[v1.HighlightWindowRequest],
) (*connect.Response[v1.HighlightWindowResponse], error) {
	if h.state.TabLinks == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoTabLinks)
	}
	h.state.TabLinks.Highlight(r.Msg.GetWindowId(), r.Msg.GetIsOn())
	return connect.NewResponse(&v1.HighlightWindowResponse{}), nil
}

func convertMoveErrorToWire(outcome tablinks.MoveError) v1.TabMoveErrorCode {
	switch outcome {
	case tablinks.MoveErrorWindowClosed:
		return v1.TabMoveErrorCode_TAB_MOVE_ERROR_CODE_WINDOW_CLOSED
	case tablinks.MoveErrorTimeout:
		return v1.TabMoveErrorCode_TAB_MOVE_ERROR_CODE_TIMEOUT
	default:
		return v1.TabMoveErrorCode_TAB_MOVE_ERROR_CODE_UNSPECIFIED
	}
}

func convertMovedTabFromWire(tab *v1.Tab) tablinks.MovedTab {
	return tablinks.MovedTab{
		ID:    tab.GetId(),
		Kind:  tab.GetKind(),
		Path:  tab.GetPath(),
		Title: tab.GetTitle(),
		State: tab.GetState(),
	}
}

func convertMovedTabToWire(tab tablinks.MovedTab) *v1.Tab {
	return &v1.Tab{Id: tab.ID, Kind: tab.Kind, Path: tab.Path, Title: tab.Title, State: tab.State}
}
