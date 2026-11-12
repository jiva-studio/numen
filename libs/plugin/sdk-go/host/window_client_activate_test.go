package host_test

import (
	"context"
	"errors"
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

// activateHost returns ActivateTab with a fixed result and records the requests.
type activateHost struct {
	numenv1connect.UnimplementedHostWindowServiceHandler
	code         v1.HostWindowErrorCode
	status       *connect.Error
	authHeader   string
	requestedIDs []string
}

func (h *activateHost) ActivateTab(_ context.Context, req *connect.Request[v1.ActivateTabRequest]) (*connect.Response[v1.ActivateTabResponse], error) {
	h.authHeader = req.Header().Get("Authorization")
	h.requestedIDs = append(h.requestedIDs, req.Msg.GetTabId())
	if h.status != nil {
		return nil, h.status
	}
	if h.code != v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_UNSPECIFIED {
		return connect.NewResponse(&v1.ActivateTabResponse{Error: h.code}), nil
	}
	return connect.NewResponse(&v1.ActivateTabResponse{
		Tab:      &v1.OpenTab{Id: req.Msg.GetTabId(), Kind: "reader:reader-pdf", Path: "library/Birds.pdf", Title: "Birds", Place: &v1.TabPlace{Page: 12, PageCount: 80}},
		ActiveId: "tab-1",
	}), nil
}

func dialActivate(t *testing.T, h *activateHost) *host.Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostWindowServiceHandler(h))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return host.NewClient(hostcall.Endpoint{BaseURL: server.URL, Token: "tok"})
}

func TestClientActivateTabReturnsTheTabWithItsPlaceAndTheActiveID(t *testing.T) {
	h := &activateHost{}
	c := dialActivate(t, h)

	tab, activeID, err := c.ActivateTab(t.Context(), "tab-4")
	if err != nil {
		t.Fatal(err)
	}
	want := host.OpenTab{ID: "tab-4", Kind: "reader:reader-pdf", Path: "library/Birds.pdf", Title: "Birds", Place: &host.TabPlace{Page: 12, PageCount: 80}}
	if !reflect.DeepEqual(tab, want) || activeID != "tab-1" {
		t.Errorf("returned %s with %q active, want %s with tab-1", formatTabs([]host.OpenTab{tab}), activeID, formatTabs([]host.OpenTab{want}))
	}
	if len(h.requestedIDs) != 1 || h.requestedIDs[0] != "tab-4" || h.authHeader != "Bearer tok" {
		t.Errorf("the host received %v with %q, want [tab-4] with Bearer tok", h.requestedIDs, h.authHeader)
	}
}

func TestClientActivateTabReturnsTheErrorsOfTheHost(t *testing.T) {
	for name, tc := range map[string]struct {
		h    *activateHost
		want error
	}{
		"no tab with the id": {&activateHost{code: v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_NOT_FOUND}, host.ErrNotExist},
		"no window":          {&activateHost{code: v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_NO_WINDOW}, host.ErrNoWindow},
		"permission denied": {
			&activateHost{status: connect.NewError(connect.CodePermissionDenied, errors.New("plugin lacks window:focus"))}, host.ErrPermissionDenied,
		},
	} {
		t.Run(name, func(t *testing.T) {
			c := dialActivate(t, tc.h)
			if _, _, err := c.ActivateTab(t.Context(), "tab-9"); !errors.Is(err, tc.want) {
				t.Errorf("ActivateTab got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestClientActivateTabReturnsErrorWhenTabNotShownInTime(t *testing.T) {
	h := &activateHost{status: connect.NewError(connect.CodeDeadlineExceeded, errors.New(`the window did not show the tab "tab-4" in time`))}
	c := dialActivate(t, h)

	tab, activeID, err := c.ActivateTab(t.Context(), "tab-4")
	if err == nil {
		t.Fatalf("ActivateTab returned %+v with %q active, want an error", tab, activeID)
	}
	if connect.CodeOf(err) != connect.CodeDeadlineExceeded || !strings.Contains(err.Error(), `"tab-4"`) {
		t.Errorf("ActivateTab got %v, want the deadline the host returned, naming tab-4", err)
	}
}
