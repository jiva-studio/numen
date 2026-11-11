package host_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

// windowHost returns HostWindowService with fixed values and records the requests.
type windowHost struct {
	numenv1connect.UnimplementedHostWindowServiceHandler
	code       v1.HostWindowErrorCode
	status     *connect.Error
	authHeader string
	notes      []*v1.FocusNoteRequest
	passages   []*v1.FocusPassagesRequest
}

func (h *windowHost) FocusNote(_ context.Context, req *connect.Request[v1.FocusNoteRequest]) (*connect.Response[v1.FocusNoteResponse], error) {
	h.authHeader = req.Header().Get("Authorization")
	h.notes = append(h.notes, req.Msg)
	if h.status != nil {
		return nil, h.status
	}
	if h.code != v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_UNSPECIFIED {
		return connect.NewResponse(&v1.FocusNoteResponse{Error: h.code}), nil
	}
	return connect.NewResponse(&v1.FocusNoteResponse{Path: req.Msg.GetPath(), Title: "Gray Heron", Id: "01J8HERON0000000000000000"}), nil
}

func (h *windowHost) FocusPassages(_ context.Context, req *connect.Request[v1.FocusPassagesRequest]) (*connect.Response[v1.FocusPassagesResponse], error) {
	h.authHeader = req.Header().Get("Authorization")
	h.passages = append(h.passages, req.Msg)
	if h.status != nil {
		return nil, h.status
	}
	if h.code != v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_UNSPECIFIED {
		return connect.NewResponse(&v1.FocusPassagesResponse{Error: h.code}), nil
	}
	return connect.NewResponse(&v1.FocusPassagesResponse{}), nil
}

func (h *windowHost) ReadOpenTabs(_ context.Context, req *connect.Request[v1.HostWindowServiceReadOpenTabsRequest]) (*connect.Response[v1.HostWindowServiceReadOpenTabsResponse], error) {
	h.authHeader = req.Header().Get("Authorization")
	if h.status != nil {
		return nil, h.status
	}
	if h.code != v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_UNSPECIFIED {
		return connect.NewResponse(&v1.HostWindowServiceReadOpenTabsResponse{Error: h.code}), nil
	}
	return connect.NewResponse(&v1.HostWindowServiceReadOpenTabsResponse{
		ActiveId: "tab-2",
		Tabs: []*v1.OpenTab{
			{Id: "tab-1", Kind: "note", Path: "notes/Heron.md", Title: "Gray Heron"},
			{Id: "tab-2", Kind: "reader:reader-pdf", Path: "library/Birds.pdf", Title: "Birds", Place: &v1.TabPlace{Page: 4, PageCount: 80}},
			{Id: "tab-3", Kind: "recording", Path: "talks/Heron Talk.mp3", Title: "Heron Talk", Place: &v1.TabPlace{PositionSeconds: new(float64)}},
			{Id: "tab-4", Kind: "reader:reader-epub", Path: "library/Marsh.epub", Title: "Marsh", Place: &v1.TabPlace{Cfi: "epubcfi(/6/4!/4/2/1:0)", Chapter: "The Reed Beds"}},
		},
	}), nil
}

func dialWindow(t *testing.T, h *windowHost) *host.Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostWindowServiceHandler(h))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return host.NewClient(hostcall.Endpoint{BaseURL: server.URL, Token: "tok"})
}

func TestClientFocusesANoteAndReadsWhatTheHostFocused(t *testing.T) {
	h := &windowHost{}
	c := dialWindow(t, h)

	got, err := c.FocusNote(t.Context(), "notes/Heron.md")
	if err != nil {
		t.Fatal(err)
	}
	want := host.FocusedNote{Path: "notes/Heron.md", Title: "Gray Heron", ID: "01J8HERON0000000000000000"}
	if got != want {
		t.Errorf("focused %+v, want %+v", got, want)
	}
	if len(h.notes) != 1 || h.notes[0].GetPath() != "notes/Heron.md" || h.authHeader != "Bearer tok" {
		t.Errorf("the host received %v with %q", h.notes, h.authHeader)
	}
}

func TestClientSendsThePassagesInTheirOrder(t *testing.T) {
	h := &windowHost{}
	c := dialWindow(t, h)

	if err := c.FocusPassages(t.Context(), "library/Birds.pdf", []host.PassageSpan{{Start: 1500, Length: 80}, {Start: 400, Length: 700}}); err != nil {
		t.Fatal(err)
	}
	if len(h.passages) != 1 {
		t.Fatalf("the host received %d times, want once", len(h.passages))
	}
	r := h.passages[0]
	spans := r.GetPassages()
	if r.GetPath() != "library/Birds.pdf" || len(spans) != 2 ||
		spans[0].GetStart() != 1500 || spans[0].GetLength() != 80 || spans[1].GetStart() != 400 || spans[1].GetLength() != 700 {
		t.Errorf("the host received %v", r)
	}
}

func TestClientReadsTheOpenTabsWithThePlaceEachReaderReported(t *testing.T) {
	h := &windowHost{}
	c := dialWindow(t, h)

	tabs, activeID, err := c.ReadOpenTabs(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	atStart := 0.0
	want := []host.OpenTab{
		{ID: "tab-1", Kind: "note", Path: "notes/Heron.md", Title: "Gray Heron"},
		{ID: "tab-2", Kind: "reader:reader-pdf", Path: "library/Birds.pdf", Title: "Birds", Place: &host.TabPlace{Page: 4, PageCount: 80}},
		{ID: "tab-3", Kind: "recording", Path: "talks/Heron Talk.mp3", Title: "Heron Talk", Place: &host.TabPlace{PositionSeconds: &atStart}},
		{ID: "tab-4", Kind: "reader:reader-epub", Path: "library/Marsh.epub", Title: "Marsh", Place: &host.TabPlace{CFI: "epubcfi(/6/4!/4/2/1:0)", Chapter: "The Reed Beds"}},
	}
	if !reflect.DeepEqual(tabs, want) || activeID != "tab-2" {
		t.Errorf("read %s with %q active, want %s with tab-2", formatTabs(tabs), activeID, formatTabs(want))
	}
}

func formatTabs(tabs []host.OpenTab) string {
	var b strings.Builder
	for _, tab := range tabs {
		fmt.Fprintf(&b, "%+v", tab)
		if tab.Place != nil {
			fmt.Fprintf(&b, " at %+v", *tab.Place)
			if tab.Place.PositionSeconds != nil {
				fmt.Fprintf(&b, " (%v s)", *tab.Place.PositionSeconds)
			}
		}
		b.WriteString("; ")
	}
	return b.String()
}

func TestClientReturnsTheErrorsOfTheWindowCalls(t *testing.T) {
	for name, tc := range map[string]struct {
		h    *windowHost
		want error
	}{
		"no window":         {&windowHost{code: v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_NO_WINDOW}, host.ErrNoWindow},
		"not found":         {&windowHost{code: v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_NOT_FOUND}, host.ErrNotExist},
		"permission denied": {&windowHost{status: connect.NewError(connect.CodePermissionDenied, errors.New("plugin lacks window:focus"))}, host.ErrPermissionDenied},
	} {
		t.Run(name, func(t *testing.T) {
			c := dialWindow(t, tc.h)
			ctx := t.Context()

			if _, err := c.FocusNote(ctx, "notes/Heron.md"); !errors.Is(err, tc.want) {
				t.Errorf("FocusNote got %v, want %v", err, tc.want)
			}
			if err := c.FocusPassages(ctx, "library/Birds.pdf", nil); !errors.Is(err, tc.want) {
				t.Errorf("FocusPassages got %v, want %v", err, tc.want)
			}
			if errors.Is(tc.want, host.ErrNotExist) {
				return
			}
			if _, _, err := c.ReadOpenTabs(ctx); !errors.Is(err, tc.want) {
				t.Errorf("ReadOpenTabs got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestClientReturnsAnInvalidPassageAsTheHostRejectedIt(t *testing.T) {
	h := &windowHost{status: connect.NewError(connect.CodeInvalidArgument, errors.New("highlight at most 8 places of one document"))}
	c := dialWindow(t, h)

	err := c.FocusPassages(t.Context(), "library/Birds.pdf", []host.PassageSpan{{Start: -1, Length: 4}})
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("got %v, want the invalid argument the host returned", err)
	}
}

func TestThePermissionConstantsAreThePermissionStrings(t *testing.T) {
	for got, want := range map[string]string{
		host.PermissionIndexRead:    "index:read",
		host.PermissionSearchPolicy: "search:policy",
		host.PermissionVaultManage:  "vault:manage",
		host.PermissionWindowFocus:  "window:focus",
		host.PermissionLinkOpen:     "link:open",
	} {
		if got != want {
			t.Errorf("the permission constant is %q, want %q", got, want)
		}
	}
}
