package workspace

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/tablink"
)

// ReadDetachedTab returns the tab of a pending detach, with the state its kind
// saved, to the first window that reads it.
func (h *Handler) ReadDetachedTab(
	_ context.Context, r *connect.Request[v1.ReadDetachedTabRequest],
) (*connect.Response[v1.ReadDetachedTabResponse], error) {
	out := &v1.ReadDetachedTabResponse{}
	if tab, moveID, ok := h.links.Transfers.ReadDetached(r.Msg.GetWindowId(), r.Msg.GetTabId()); ok {
		out.Tab = convertSavedTabToWire(tab)
		out.MoveId = moveID
	}
	return connect.NewResponse(out), nil
}

// MoveTab sends a tab to another window and responds after the target window
// accepts it, its watch closes, or the move times out. A canceled request ends
// the move, so the target window cannot accept it afterwards.
func (h *Handler) MoveTab(
	ctx context.Context, r *connect.Request[v1.MoveTabRequest],
) (*connect.Response[v1.MoveTabResponse], error) {
	if r.Msg.GetWindowId() == "" || r.Msg.GetTargetWindowId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNoWindowID)
	}
	tab := convertSavedTabFromWire(r.Msg.GetTab())
	result := h.links.Transfers.Move(r.Msg.GetWindowId(), r.Msg.GetTargetWindowId(), tab)
	select {
	case <-ctx.Done():
		h.links.Transfers.CancelMove(tab.ID, result)
		return nil, connect.NewError(connect.CodeCanceled, ctx.Err())
	case outcome := <-result:
		return connect.NewResponse(&v1.MoveTabResponse{Error: convertMoveErrorToWire(outcome)}), nil
	}
}

// AcceptTabMove completes the move of a tab into the window that opened it.
func (h *Handler) AcceptTabMove(
	_ context.Context, r *connect.Request[v1.AcceptTabMoveRequest],
) (*connect.Response[v1.AcceptTabMoveResponse], error) {
	outcome := h.links.Transfers.Accept(r.Msg.GetWindowId(), r.Msg.GetTabId(), r.Msg.GetMoveId())
	return connect.NewResponse(&v1.AcceptTabMoveResponse{Error: convertMoveErrorToWire(outcome)}), nil
}

// HighlightWindow turns the outline of a window on or off.
func (h *Handler) HighlightWindow(
	_ context.Context, r *connect.Request[v1.HighlightWindowRequest],
) (*connect.Response[v1.HighlightWindowResponse], error) {
	h.links.Windows.Highlight(r.Msg.GetWindowId(), r.Msg.GetIsOn())
	return connect.NewResponse(&v1.HighlightWindowResponse{}), nil
}

func convertMoveErrorToWire(outcome tablink.MoveError) v1.TabMoveErrorCode {
	switch outcome {
	case tablink.MoveErrorWindowClosed:
		return v1.TabMoveErrorCode_TAB_MOVE_ERROR_CODE_WINDOW_CLOSED
	case tablink.MoveErrorTimeout:
		return v1.TabMoveErrorCode_TAB_MOVE_ERROR_CODE_TIMEOUT
	case tablink.MoveErrorNotOwner:
		return v1.TabMoveErrorCode_TAB_MOVE_ERROR_CODE_NOT_OWNER
	case tablink.MoveErrorCanceled:
		return v1.TabMoveErrorCode_TAB_MOVE_ERROR_CODE_CANCELED
	default:
		return v1.TabMoveErrorCode_TAB_MOVE_ERROR_CODE_UNSPECIFIED
	}
}

func convertSavedTabFromWire(tab *v1.Tab) tablink.SavedTab {
	return tablink.SavedTab{
		ID:    tab.GetId(),
		Kind:  tab.GetKind(),
		Path:  tab.GetPath(),
		Title: tab.GetTitle(),
		State: tab.GetState(),
	}
}

func convertSavedTabToWire(tab tablink.SavedTab) *v1.Tab {
	return &v1.Tab{Id: tab.ID, Kind: tab.Kind, Path: tab.Path, Title: tab.Title, State: tab.State}
}
