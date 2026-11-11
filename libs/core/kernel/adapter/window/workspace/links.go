package workspace

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/tablink"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/tablinks"
)

// LinkGrace is how long a detached tab, and the tabs of a window whose last
// watch ended, keep their links before a window reports them.
const LinkGrace = 30 * time.Second

var errNoWindowID = errors.New("a window id is required")

// WatchTabLinks streams the tab links to a window until the window cancels the
// watch or disconnects. When the last stream of the window ends, its tabs are
// unlinked unless the window watches again within the grace period.
func (h *Handler) WatchTabLinks(
	ctx context.Context,
	r *connect.Request[v1.WatchTabLinksRequest],
	out *connect.ServerStream[v1.WatchTabLinksResponse],
) error {
	if r.Msg.GetWindowId() == "" {
		return connect.NewError(connect.CodeInvalidArgument, errNoWindowID)
	}
	watch := h.links.Windows.Watch(r.Msg.GetWindowId())
	defer watch.Close()

	if err := out.Send(convertSnapshotToWire(watch.Initial)); err != nil {
		return err
	}

	repeat := time.NewTicker(wire.ResendInterval)
	defer repeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-watch.Stop:
			return nil
		case <-repeat.C:
			// An empty response is a keepalive; its send fails once the client disconnects.
			if err := out.Send(&v1.WatchTabLinksResponse{}); err != nil {
				return err
			}
		case event, open := <-watch.Events:
			if !open {
				return nil
			}
			if err := sendTabLinksEvent(out, r.Msg.GetWindowId(), event); err != nil {
				return err
			}
		}
	}
}

func sendTabLinksEvent(out *connect.ServerStream[v1.WatchTabLinksResponse], windowID string, event tablinks.Event) error {
	if !event.IsFor(windowID) {
		return nil
	}
	return out.Send(convertEventToWire(event))
}

// CancelWatchTabLinks cancels the WatchTabLinks streams of a window.
func (h *Handler) CancelWatchTabLinks(
	_ context.Context, r *connect.Request[v1.CancelWatchTabLinksRequest],
) (*connect.Response[v1.CancelWatchTabLinksResponse], error) {
	h.links.Windows.Cancel(r.Msg.GetWindowId())
	return connect.NewResponse(&v1.CancelWatchTabLinksResponse{}), nil
}

// WriteTabLinks applies a link, an unlink or a tab report from a window.
func (h *Handler) WriteTabLinks(
	_ context.Context, r *connect.Request[v1.WriteTabLinksRequest],
) (*connect.Response[v1.WriteTabLinksResponse], error) {
	switch change := r.Msg.GetChange().(type) {
	case *v1.WriteTabLinksRequest_Link:
		h.links.Links.Link(change.Link.GetSourceTabId(), change.Link.GetTargetTabId())
	case *v1.WriteTabLinksRequest_UnlinkTabId:
		h.links.Links.Unlink(change.UnlinkTabId)
	case *v1.WriteTabLinksRequest_Tabs:
		if r.Msg.GetWindowId() == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errNoWindowID)
		}
		h.links.Windows.WriteTabs(
			convertWindowFromWire(r.Msg.GetWindowId(), change.Tabs.GetWindow()),
			convertTabsFromWire(change.Tabs.GetTabs()),
		)
	}
	return connect.NewResponse(&v1.WriteTabLinksResponse{}), nil
}

// SyncTabPath sends the new path of a linked tab to every watching window.
func (h *Handler) SyncTabPath(
	_ context.Context, r *connect.Request[v1.SyncTabPathRequest],
) (*connect.Response[v1.SyncTabPathResponse], error) {
	h.links.Links.SyncPath(r.Msg.GetSourceTabId(), r.Msg.GetPath(), r.Msg.GetTitle())
	return connect.NewResponse(&v1.SyncTabPathResponse{}), nil
}

func convertTabsFromWire(tabs []*v1.LinkableTab) []tablink.Tab {
	out := make([]tablink.Tab, 0, len(tabs))
	for _, one := range tabs {
		out = append(out, tablink.Tab{ID: one.GetId(), Kind: one.GetKind(), Title: one.GetTitle()})
	}
	return out
}

func convertWindowFromWire(windowID string, window *v1.TabWindow) tablinks.Window {
	return tablinks.Window{
		ID:          windowID,
		IsMain:      window.GetIsMain(),
		ActiveTitle: window.GetActiveTitle(),
		ActiveKind:  window.GetActiveKind(),
		TabCount:    int(window.GetTabCount()),
	}
}

func convertEventToWire(event tablinks.Event) *v1.WatchTabLinksResponse {
	switch {
	case event.Sync != nil:
		return &v1.WatchTabLinksResponse{Event: &v1.WatchTabLinksResponse_Sync{Sync: &v1.TabPathSync{
			GroupId:     event.Sync.GroupID,
			SourceTabId: event.Sync.SourceTabID,
			Path:        event.Sync.Path,
			Title:       event.Sync.Title,
		}}}
	case event.Move != nil:
		return &v1.WatchTabLinksResponse{Event: &v1.WatchTabLinksResponse_MovedTab{
			MovedTab: &v1.MovedTab{Tab: convertSavedTabToWire(event.Move.Tab), MoveId: event.Move.MoveID},
		}}
	case event.Highlight != nil:
		return &v1.WatchTabLinksResponse{Event: &v1.WatchTabLinksResponse_Highlight{
			Highlight: &v1.WindowHighlight{IsOn: event.Highlight.IsOn},
		}}
	case event.Snapshot != nil:
		return convertSnapshotToWire(*event.Snapshot)
	default:
		return &v1.WatchTabLinksResponse{}
	}
}

func convertSnapshotToWire(snapshot tablinks.Snapshot) *v1.WatchTabLinksResponse {
	links := &v1.TabLinks{
		Groups: make([]*v1.TabLinkGroup, 0, len(snapshot.Groups)),
		Tabs:   make([]*v1.LinkableTab, 0, len(snapshot.Tabs)),
	}
	for _, group := range snapshot.Groups {
		links.Groups = append(links.Groups, &v1.TabLinkGroup{
			Id:         group.ID,
			ColorIndex: int32(group.ColorIndex),
			TabIds:     group.TabIDs,
			Path:       group.Path,
			Title:      group.Title,
		})
	}
	for _, tab := range snapshot.Tabs {
		links.Tabs = append(links.Tabs, &v1.LinkableTab{
			Id: tab.ID, Kind: tab.Kind, Title: tab.Title, WindowId: tab.WindowID,
		})
	}
	for _, window := range snapshot.Windows {
		links.Windows = append(links.Windows, &v1.TabWindow{
			Id:          window.ID,
			IsMain:      window.IsMain,
			ActiveTitle: window.ActiveTitle,
			ActiveKind:  window.ActiveKind,
			TabCount:    int32(window.TabCount),
		})
	}
	return &v1.WatchTabLinksResponse{Event: &v1.WatchTabLinksResponse_Links{Links: links}}
}
