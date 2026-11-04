package editor

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/tablinks"
	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
)

// detachGrace is how long a detached tab keeps its links before a window
// reports it.
const detachGrace = 30 * time.Second

var (
	errNoTabLinks = errors.New("this build serves no tab links")
	errNoWindowID = errors.New("a window id is required")
)

// WatchTabLinks streams the tab links to a window until the window cancels the
// watch or disconnects. The tabs of the window leave their groups when it ends.
func (h *workspaceHandler) WatchTabLinks(
	ctx context.Context,
	r *connect.Request[v1.WatchTabLinksRequest],
	out *connect.ServerStream[v1.WatchTabLinksResponse],
) error {
	if h.state.TabLinks == nil {
		return connect.NewError(connect.CodeUnimplemented, errNoTabLinks)
	}
	if r.Msg.GetWindowId() == "" {
		return connect.NewError(connect.CodeInvalidArgument, errNoWindowID)
	}
	watch := h.state.TabLinks.Watch(r.Msg.GetWindowId())
	defer watch.Close()

	if err := out.Send(convertSnapshotToWire(watch.Initial)); err != nil {
		return err
	}

	repeat := time.NewTicker(wire.Again)
	defer repeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-watch.Done:
			return nil
		case <-repeat.C:
			// An empty response is a keepalive that fails when the client has disconnected.
			if err := out.Send(&v1.WatchTabLinksResponse{}); err != nil {
				return err
			}
		case event, open := <-watch.Events:
			if !open {
				return nil
			}
			if err := out.Send(convertEventToWire(event)); err != nil {
				return err
			}
		}
	}
}

// CancelWatchTabLinks ends the WatchTabLinks streams of a window.
func (h *workspaceHandler) CancelWatchTabLinks(
	_ context.Context, r *connect.Request[v1.CancelWatchTabLinksRequest],
) (*connect.Response[v1.CancelWatchTabLinksResponse], error) {
	if h.state.TabLinks == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoTabLinks)
	}
	h.state.TabLinks.Cancel(r.Msg.GetWindowId())
	return connect.NewResponse(&v1.CancelWatchTabLinksResponse{}), nil
}

// WriteTabLinks applies one change of a window to the tab links.
func (h *workspaceHandler) WriteTabLinks(
	_ context.Context, r *connect.Request[v1.WriteTabLinksRequest],
) (*connect.Response[v1.WriteTabLinksResponse], error) {
	if h.state.TabLinks == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoTabLinks)
	}
	switch change := r.Msg.GetChange().(type) {
	case *v1.WriteTabLinksRequest_Link:
		h.state.TabLinks.Link(change.Link.GetSourceTabId(), change.Link.GetTargetTabId())
	case *v1.WriteTabLinksRequest_UnlinkTabId:
		h.state.TabLinks.Unlink(change.UnlinkTabId)
	case *v1.WriteTabLinksRequest_Tabs:
		if r.Msg.GetWindowId() == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errNoWindowID)
		}
		h.state.TabLinks.WriteTabs(r.Msg.GetWindowId(), convertTabsFromWire(change.Tabs.GetTabs()))
	}
	return connect.NewResponse(&v1.WriteTabLinksResponse{}), nil
}

// SyncTabPath sends the path a linked tab moved to to every watching window.
func (h *workspaceHandler) SyncTabPath(
	_ context.Context, r *connect.Request[v1.SyncTabPathRequest],
) (*connect.Response[v1.SyncTabPathResponse], error) {
	if h.state.TabLinks == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoTabLinks)
	}
	h.state.TabLinks.SyncPath(r.Msg.GetSourceTabId(), r.Msg.GetPath(), r.Msg.GetTitle())
	return connect.NewResponse(&v1.SyncTabPathResponse{}), nil
}

func convertTabsFromWire(tabs []*v1.LinkableTab) []tablinks.Tab {
	out := make([]tablinks.Tab, 0, len(tabs))
	for _, one := range tabs {
		out = append(out, tablinks.Tab{ID: one.GetId(), Kind: one.GetKind(), Title: one.GetTitle()})
	}
	return out
}

func convertEventToWire(event tablinks.Event) *v1.WatchTabLinksResponse {
	if event.Sync != nil {
		return &v1.WatchTabLinksResponse{Event: &v1.WatchTabLinksResponse_Sync{Sync: &v1.TabPathSync{
			GroupId:     event.Sync.GroupID,
			SourceTabId: event.Sync.SourceTabID,
			Path:        event.Sync.Path,
			Title:       event.Sync.Title,
		}}}
	}
	if event.Snapshot != nil {
		return convertSnapshotToWire(*event.Snapshot)
	}
	return &v1.WatchTabLinksResponse{}
}

func convertSnapshotToWire(snapshot tablinks.Snapshot) *v1.WatchTabLinksResponse {
	links := &v1.TabLinks{
		Groups: make([]*v1.TabLinkGroup, 0, len(snapshot.Groups)),
		Tabs:   make([]*v1.LinkableTab, 0, len(snapshot.Tabs)),
	}
	for _, group := range snapshot.Groups {
		links.Groups = append(links.Groups, &v1.TabLinkGroup{
			Id: group.ID, ColorIndex: int32(group.ColorIndex), TabIds: group.TabIDs,
		})
	}
	for _, tab := range snapshot.Tabs {
		links.Tabs = append(links.Tabs, &v1.LinkableTab{
			Id: tab.ID, Kind: tab.Kind, Title: tab.Title, WindowId: tab.WindowID,
		})
	}
	return &v1.WatchTabLinksResponse{Event: &v1.WatchTabLinksResponse_Links{Links: links}}
}
