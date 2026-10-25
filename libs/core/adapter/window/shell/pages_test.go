package shell

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/internal/csp"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
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
	api := newAPI(t)
	api.Themes = dressing{wearing: "Evening"}

	server := httptest.NewServer(api.NewHandler(http.NotFoundHandler()))
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

// A folder of themes that could not be made leaves the window undressed and not unopened: it
// goes on answering, and asking about themes is what fails.
func TestAWindowWithNoThemesStillAnswers(t *testing.T) {
	api := newAPI(t)

	server := httptest.NewServer(api.NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)

	window := numenv1connect.NewWindowServiceClient(server.Client(), server.URL)
	stream, err := window.WatchTasks(t.Context(), connect.NewRequest(&v1.WatchTasksRequest{Window: testWindow}))
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

// The page itself is asked for at its addresses, and a build carrying no
// interface answers them with the files the window was given.
func TestThePageIsAnsweredAtEveryAddressItOpensAt(t *testing.T) {
	files := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("the files"))
	})
	handler := (&API{}).NewHandler(files)

	_, err := wire.GetInterface(pages)
	for _, item := range []string{"/", "/index.html"} {
		out := httptest.NewRecorder()
		handler.ServeHTTP(out, httptest.NewRequest(http.MethodGet, item, nil))
		if err == nil {
			if !strings.Contains(out.Body.String(), "<!doctype html>") {
				t.Errorf("%s answered %q", item, out.Body.String())
			}
		} else {
			if got := out.Body.String(); got != "the files" {
				t.Errorf("%s answered %q", item, got)
			}
		}
	}
}

// A `data:` URI is the whole of what this window widens: no script from elsewhere, no form submitted
// anywhere, and nothing fetched off the machine.
func TestTheWindowIsHeldToOnePolicy(t *testing.T) {
	target := csp.Sources{Images: []string{"data:"}}.Policy()
	if policy != target {
		t.Errorf("the policy reads %q", policy)
	}

	handler := (&API{}).NewHandler(http.NotFoundHandler())
	for _, path := range []string{"", "/", "/index.html", "/built/index.css"} {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.URL.Path = path
		out := httptest.NewRecorder()
		handler.ServeHTTP(out, r)
		if response := out.Header().Get("Content-Security-Policy"); response != target {
			t.Errorf("%q is bound to %q", path, response)
		}
	}
}

// A call for a service a plugin serves is forwarded to the plugin, and a path no plugin serves falls
// through to the files.
func TestTheServicesPluginsServeAreForwarded(t *testing.T) {
	forwarded := &forwarding{serves: map[string]bool{"/numen.v1.BookService/GetBook": true}}
	api := &API{PluginServices: forwarded}
	handler := api.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("the files"))
	}))

	for path, want := range map[string]string{
		"/numen.v1.BookService/GetBook": "a plugin",
		"/built/index.css":              "the files",
	} {
		out := httptest.NewRecorder()
		handler.ServeHTTP(out, httptest.NewRequest(http.MethodPost, path, nil))
		if got := out.Body.String(); got != want {
			t.Errorf("%s answered %q, want %q", path, got, want)
		}
	}
}

// forwarding is plugins serving the services it names.
type forwarding struct{ serves map[string]bool }

func (f *forwarding) Serves(path string) bool { return f.serves[path] }

func (f *forwarding) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("a plugin"))
}

// The window lists the plugins of its target and serves the files of their frontends from the folder
// each is installed in.
func TestTheWindowServesItsPluginsAndTheirFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "dist"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "dist", "index.js"), []byte("export {}"), 0o600); err != nil {
		t.Fatal(err)
	}
	api := newAPI(t)
	api.GetPluginIDs = func() []string { return []string{"md.numen.example"} }
	api.GetPluginDir = func(id string) (string, bool) { return dir, id == "md.numen.example" }
	server := httptest.NewServer(api.NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)

	plugins := numenv1connect.NewPluginServiceClient(server.Client(), server.URL)
	out, err := plugins.GetPlugins(t.Context(), connect.NewRequest(&v1.GetPluginsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if got := out.Msg.GetPlugins(); len(got) != 1 || got[0].GetId() != "md.numen.example" {
		t.Errorf("the window lists %v", got)
	}

	for path, want := range map[string]int{
		"/api/plugins/md.numen.example/dist/index.js": http.StatusOK,
		"/api/plugins/md.numen.other/dist/index.js":   http.StatusNotFound,
	} {
		response, err := server.Client().Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		_ = response.Body.Close()
		if response.StatusCode != want {
			t.Errorf("%s answered %d, want %d", path, response.StatusCode, want)
		}
	}
}

// A window given no plugins lists none.
func TestAWindowWithNoPluginsListsNone(t *testing.T) {
	server := httptest.NewServer(newAPI(t).NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)

	plugins := numenv1connect.NewPluginServiceClient(server.Client(), server.URL)
	out, err := plugins.GetPlugins(t.Context(), connect.NewRequest(&v1.GetPluginsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if got := out.Msg.GetPlugins(); len(got) != 0 {
		t.Errorf("a window with no plugins lists %v", got)
	}
}
