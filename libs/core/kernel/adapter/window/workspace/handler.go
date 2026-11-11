// Package workspace answers the workspace service of the editor window: the tabs the window has
// open, the places it is asked to show, and the links and moves of tabs between windows.
package workspace

import (
	"context"
	"errors"
	"math"
	"net/http"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/tablinks"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// Window is the window the tabs belong to: the vault it shows, the tabs it reports, and where it
// is asked to focus.
type Window interface {
	GetCurrentVault() domain.Vault
	// ReportOpenTabs keeps the tabs the window reported as the ones it has open.
	ReportOpenTabs(open window.OpenTabs)
}

// Focus delivers the requests to show a place or to bring a tab forward.
type Focus interface {
	SubscribePlaces() (<-chan domain.Place, func())
	SubscribeTabIDs() (<-chan string, func())
}

// TabWindows opens a tab in a window of its own, in a build that can.
type TabWindows interface {
	CanOpenTabWindow() bool
	OpenTabWindow(ctx context.Context, tab window.DetachedTab) error
}

// OpenTabsWriter stores the open tabs of a vault.
type OpenTabsWriter interface {
	Execute(ctx context.Context, v domain.Vault, open window.OpenTabs) error
}

// OpenTabsReader loads the open tabs stored for a vault.
type OpenTabsReader interface {
	Execute(ctx context.Context, v domain.Vault) (window.OpenTabs, error)
}

// Handler answers the window's workspace state: the open tabs, the focus requests and the tab links.
type Handler struct {
	window     Window
	focus      Focus
	tabWindows TabWindows
	write      OpenTabsWriter
	read       OpenTabsReader
	links      *tablinks.Tabs
}

func NewHandler(
	current Window, focus Focus, tabWindows TabWindows, write OpenTabsWriter, read OpenTabsReader, links *tablinks.Tabs,
) *Handler {
	return &Handler{window: current, focus: focus, tabWindows: tabWindows, write: write, read: read, links: links}
}

// NewLinks is the registry of the tab links of one window.
func NewLinks() (*tablinks.Tabs, error) {
	return tablinks.New(LinkGrace, scheduleAfter, newLinkEvents())
}

// NewRoute is where the workspace service answers: its path and the handler of its calls. Each call
// goes through counted.
func NewRoute(handler *Handler, counted connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewWorkspaceServiceHandler(handler, counted)
}

// WriteOpenTabs stores the currently open tabs, as last reported by the window,
// for the vault in the window.
//
// It is the reverse direction of WatchFocus: WatchFocus delivers a focus request
// to the window, and WriteOpenTabs reports the window's current tabs.
func (h *Handler) WriteOpenTabs(
	ctx context.Context, r *connect.Request[v1.WriteOpenTabsRequest],
) (*connect.Response[v1.WriteOpenTabsResponse], error) {
	tabs := r.Msg.GetTabs()
	open := window.OpenTabs{Tabs: make([]window.Tab, 0, len(tabs)), ActiveID: r.Msg.GetActiveId()}
	for _, one := range tabs {
		open.Tabs = append(open.Tabs, window.Tab{
			ID:    one.GetId(),
			Kind:  one.GetKind(),
			Path:  one.GetPath(),
			Title: one.GetTitle(),
			State: one.GetState(),
			Place: convertTabPlaceFromWire(one.GetPlace()),
		})
	}
	h.window.ReportOpenTabs(open)
	if v := h.window.GetCurrentVault(); v.Path != "" {
		if err := h.write.Execute(ctx, v, open); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	return connect.NewResponse(&v1.WriteOpenTabsResponse{}), nil
}

// ReadOpenTabs returns the tabs kept for the vault in the window.
func (h *Handler) ReadOpenTabs(
	ctx context.Context, _ *connect.Request[v1.ReadOpenTabsRequest],
) (*connect.Response[v1.ReadOpenTabsResponse], error) {
	v := h.window.GetCurrentVault()
	if v.Path == "" {
		return connect.NewResponse(&v1.ReadOpenTabsResponse{}), nil
	}
	open, err := h.read.Execute(ctx, v)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if len(open.Tabs) == 0 && open.ActiveID == "" {
		return connect.NewResponse(&v1.ReadOpenTabsResponse{}), nil
	}
	out := &v1.ReadOpenTabsResponse{Tabs: make([]*v1.Tab, 0, len(open.Tabs)), ActiveId: open.ActiveID}
	for _, one := range open.Tabs {
		out.Tabs = append(out.Tabs, &v1.Tab{
			Id: one.ID, Kind: one.Kind, Path: one.Path, Title: one.Title, State: one.State,
		})
	}
	return connect.NewResponse(out), nil
}

// DetachTab stores a tab for ReadDetachedTab, opens a new window for it, and
// responds after that window accepts the tab or the detach ends. A canceled
// request ends the detach.
func (h *Handler) DetachTab(
	ctx context.Context, r *connect.Request[v1.DetachTabRequest],
) (*connect.Response[v1.DetachTabResponse], error) {
	if !h.tabWindows.CanOpenTabWindow() {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("tab tear-off not supported"))
	}
	tab := convertSavedTabFromWire(r.Msg.GetTab())
	if tab.ID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("a tab id is required"))
	}
	result := h.links.Transfers.Detach(tab)
	if err := h.tabWindows.OpenTabWindow(ctx, window.DetachedTab{
		ID:     tab.ID,
		X:      int(r.Msg.GetX()),
		Y:      int(r.Msg.GetY()),
		Width:  int(r.Msg.GetWidth()),
		Height: int(r.Msg.GetHeight()),
	}); err != nil {
		h.links.Transfers.CancelDetach(tab.ID)
		return nil, err
	}
	select {
	case <-ctx.Done():
		h.links.Transfers.CancelDetach(tab.ID)
		return nil, connect.NewError(connect.CodeCanceled, ctx.Err())
	case outcome := <-result:
		return connect.NewResponse(&v1.DetachTabResponse{Error: convertMoveErrorToWire(outcome)}), nil
	}
}

// WatchFocus streams focus requests to the client until it disconnects.
func (h *Handler) WatchFocus(
	ctx context.Context,
	_ *connect.Request[v1.WatchFocusRequest],
	out *connect.ServerStream[v1.WatchFocusResponse],
) error {
	places, stopPlaces := h.focus.SubscribePlaces()
	defer stopPlaces()
	tabIDs, stopTabIDs := h.focus.SubscribeTabIDs()
	defer stopTabIDs()

	repeat := time.NewTicker(wire.ResendInterval)
	defer repeat.Stop()

	for {
		msg, isOpen := receiveFocus(ctx, repeat.C, places, tabIDs)
		if !isOpen {
			return nil
		}
		if err := out.Send(msg); err != nil {
			return err
		}
	}
}

func receiveFocus(
	ctx context.Context, tick <-chan time.Time, places <-chan domain.Place, tabIDs <-chan string,
) (*v1.WatchFocusResponse, bool) {
	select {
	case <-ctx.Done():
		return nil, false
	case <-tick:
		// An empty response is a keepalive that fails when the client has disconnected.
		return &v1.WatchFocusResponse{}, true
	case id, isOpen := <-tabIDs:
		return &v1.WatchFocusResponse{TabId: id}, isOpen
	case pos, isOpen := <-places:
		return &v1.WatchFocusResponse{Path: pos.Path, Ranges: convertByteRangesToWire(pos.Ranges)}, isOpen
	}
}

func convertByteRangesToWire(pos []domain.ByteRange) []*v1.Range {
	out := make([]*v1.Range, 0, len(pos))
	for _, span := range pos {
		if span.Start < math.MinInt32 || span.Start > math.MaxInt32 || span.End < math.MinInt32 || span.End > math.MaxInt32 {
			continue
		}
		out = append(out, &v1.Range{Start: int32(span.Start), End: int32(span.End)})
	}
	return out
}

func convertTabPlaceFromWire(place *v1.TabPlace) *window.TabPlace {
	if place == nil {
		return nil
	}
	out := &window.TabPlace{
		Page:      int(place.GetPage()),
		PageCount: int(place.GetPageCount()),
		CFI:       place.GetCfi(),
		Chapter:   place.GetChapter(),
	}
	if place.PositionSeconds != nil {
		seconds := place.GetPositionSeconds()
		out.PositionSeconds = &seconds
	}
	return out
}
