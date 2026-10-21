package editor

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// workspaceHandler answers where the person is in the window: what they have open,
// and what is put in front of them.
type workspaceHandler struct{ state *State }

// WriteOpenTabs takes what currently open, as the window last said it.
//
// It is the other direction to WatchFocus: a place is put in front of the
// person there, and here the window says what is in front of them now.
func (h *workspaceHandler) WriteOpenTabs(
	_ context.Context, r *connect.Request[v1.WriteOpenTabsRequest],
) (*connect.Response[v1.WriteOpenTabsResponse], error) {
	tabs := r.Msg.GetTabs()
	open := domain.OpenTabs{Tabs: make([]domain.Tab, 0, len(tabs)), FrontID: r.Msg.GetFront()}
	for _, one := range tabs {
		tab := domain.Tab{
			ID:    one.GetId(),
			Kind:  one.GetKind(),
			Path:  one.GetPath(),
			Title: one.GetTitle(),
		}
		if doc := one.GetDocument(); doc != nil {
			tab.Document = &domain.DocumentProgress{
				Page:      int(doc.GetPage()),
				PageCount: int(doc.GetPageCount()),
			}
		}
		if book := one.GetBook(); book != nil {
			tab.Book = &domain.BookProgress{
				Offset:    int(book.GetOffset()),
				Page:      int(book.GetPage()),
				PageCount: int(book.GetPageCount()),
			}
		}
		if rec := one.GetRecording(); rec != nil {
			tab.Recording = &domain.RecordingProgress{
				TranscribedDuration: int(rec.GetTranscribedDurationMs()),
				Duration:            int(rec.GetDurationMs()),
			}
		}
		open.Tabs = append(open.Tabs, tab)
	}
	h.state.openTabs.Store(&open)
	if h.state.NotifyOpenTabs != nil {
		h.state.NotifyOpenTabs(open)
	}
	return connect.NewResponse(&v1.WriteOpenTabsResponse{}), nil
}

// DetachTab asks that a tab be opened in a new window.
func (h *workspaceHandler) DetachTab(
	ctx context.Context, r *connect.Request[v1.DetachTabRequest],
) (*connect.Response[v1.DetachTabResponse], error) {
	if h.state.DetachTab == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("tab tear-off not supported"))
	}
	if err := h.state.DetachTab(ctx, r.Msg.GetTabId(), int(r.Msg.GetX()), int(r.Msg.GetY())); err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.DetachTabResponse{}), nil
}

// GetOpenTabs is what currently open, as the window last said. A window
// that has said nothing has nothing open as far as anyone here knows.
func (s *State) GetOpenTabs() domain.OpenTabs {
	if open := s.openTabs.Load(); open != nil {
		return *open
	}
	return domain.OpenTabs{}
}

// WatchFocus reports what something else asked to be put in front of the
// person, for as long as the client listens.
func (h *workspaceHandler) WatchFocus(
	ctx context.Context,
	_ *connect.Request[v1.WatchFocusRequest],
	out *connect.ServerStream[v1.WatchFocusResponse],
) error {
	line, isDone := h.state.Places.subscribe()
	defer isDone()

	repeat := time.NewTicker(wire.Again)
	defer repeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-repeat.C:
			// A place nobody asked for is empty, and is here to fail when
			// the client has gone.
			if err := out.Send(&v1.WatchFocusResponse{}); err != nil {
				return err
			}
		case at, open := <-line:
			if !open {
				return nil
			}
			if err := out.Send(&v1.WatchFocusResponse{
				Path:  at.Path,
				Spans: convertByteSpansToWire(at.Spans),
			}); err != nil {
				return err
			}
		}
	}
}
