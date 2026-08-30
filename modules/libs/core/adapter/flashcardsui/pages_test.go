package flashcardsui

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

func (d dressing) Themes(
	context.Context, *connect.Request[v1.ThemesRequest],
) (*connect.Response[v1.ThemesResponse], error) {
	return connect.NewResponse(&v1.ThemesResponse{Applied: d.wearing}), nil
}

func (dressing) Theme(
	context.Context, *connect.Request[v1.ThemeRequest],
) (*connect.Response[v1.ThemeResponse], error) {
	return connect.NewResponse(&v1.ThemeResponse{}), nil
}

func (dressing) Choose(
	context.Context, *connect.Request[v1.ChooseRequest],
) (*connect.Response[v1.ChooseResponse], error) {
	return connect.NewResponse(&v1.ChooseResponse{}), nil
}

func (dressing) Changed(
	context.Context, *connect.Request[v1.ChangedRequest], *connect.ServerStream[v1.ChangedResponse],
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
	out, err := client.Themes(t.Context(), connect.NewRequest(&v1.ThemesRequest{}))
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
	if _, err := client.Owing(t.Context(), connect.NewRequest(&v1.OwingRequest{})); err != nil {
		t.Fatalf("a window with no themes answers nothing: %v", err)
	}

	themes := numenv1connect.NewThemeServiceClient(server.Client(), server.URL)
	if _, err := themes.Themes(t.Context(), connect.NewRequest(&v1.ThemesRequest{})); err == nil {
		t.Error("a window with no themes answered about them")
	}
}
