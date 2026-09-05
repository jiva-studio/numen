package flashcards

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// dressing is the themes an installation holds, as this window asks for them.
type dressing struct{ wearing string }

func (d dressing) ListThemes(
	context.Context, *connect.Request[v1.ListThemesRequest],
) (*connect.Response[v1.ListThemesResponse], error) {
	return connect.NewResponse(&v1.ListThemesResponse{Applied: d.wearing}), nil
}

func (dressing) ReadTheme(
	context.Context, *connect.Request[v1.ReadThemeRequest],
) (*connect.Response[v1.ReadThemeResponse], error) {
	return connect.NewResponse(&v1.ReadThemeResponse{}), nil
}

func (dressing) WriteAppearance(
	context.Context, *connect.Request[v1.WriteAppearanceRequest],
) (*connect.Response[v1.WriteAppearanceResponse], error) {
	return connect.NewResponse(&v1.WriteAppearanceResponse{}), nil
}

func (dressing) WatchThemes(
	context.Context, *connect.Request[v1.WatchThemesRequest], *connect.ServerStream[v1.WatchThemesResponse],
) error {
	return nil
}

// The window is dressed from the installation's themes, so a person who chose
// one in the editor meets it here as well.
func TestTheWindowIsDressedFromTheInstallationsThemes(t *testing.T) {
	api := &API{Registry: registry{}, Themes: dressing{wearing: "Evening"}, Now: time.Now}

	server := httptest.NewServer(api.Serving(http.NotFoundHandler()))
	t.Cleanup(server.Close)

	client := numenv1connect.NewThemeServiceClient(server.Client(), server.URL)
	out, err := client.ListThemes(t.Context(), connect.NewRequest(&v1.ListThemesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if out.Msg.GetApplied() != "Evening" {
		t.Errorf("the window is dressed in %q", out.Msg.GetApplied())
	}
}

// A folder of themes that could not be made leaves the window undressed rather
// than unopened: the page is served, and asking about themes is what fails.
func TestAWindowWithNoThemesStillServesItsPage(t *testing.T) {
	api := &API{Registry: registry{}, Now: time.Now}

	server := httptest.NewServer(api.Serving(http.NotFoundHandler()))
	t.Cleanup(server.Close)

	client := numenv1connect.NewFlashcardsServiceClient(server.Client(), server.URL)
	stream, err := client.WatchCardsDue(t.Context(), connect.NewRequest(&v1.WatchCardsDueRequest{}))
	if err != nil {
		t.Fatalf("a window with no themes answers nothing: %v", err)
	}
	t.Cleanup(func() { stream.Close() })
	if !stream.Receive() {
		t.Fatalf("a window with no themes answers nothing: %v", stream.Err())
	}

	themes := numenv1connect.NewThemeServiceClient(server.Client(), server.URL)
	if _, err := themes.ListThemes(t.Context(), connect.NewRequest(&v1.ListThemesRequest{})); err == nil {
		t.Error("a window with no themes answered about them")
	}
}
