package servicebridge

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// newRouteHost is a host running one plugin, org.example.reader, at the server given.
func newRouteHost(t *testing.T, plugin *httptest.Server) *servingHost {
	t.Helper()
	return &servingHost{
		endpoints: map[string]pluginport.Endpoint{"org.example.reader": endpointOf(t, plugin, "secret")},
		manifests: map[string]pluginport.PluginManifest{"org.example.reader": {ID: "org.example.reader"}},
	}
}

func TestServeRouteForwardsBelowThePluginRoute(t *testing.T) {
	var asked, auth, cookie string
	plugin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked, auth, cookie = r.URL.RequestURI(), r.Header.Get("Authorization"), r.Header.Get("Cookie")
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Content-Security-Policy", "default-src *")
		w.Header().Set("Set-Cookie", "plugin=1")
		if _, err := io.WriteString(w, "page bytes"); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(plugin.Close)
	host := newRouteHost(t, plugin)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/plugin/org.example.reader/pages/2?path=Books%2Fa.pdf&wide=800", nil)
	req.Header.Set("Cookie", "session=1")
	recorded := httptest.NewRecorder()

	Open(host).ServeRoute(recorded, req)

	if recorded.Code != http.StatusOK || recorded.Body.String() != "page bytes" {
		t.Fatalf("answered %d %q", recorded.Code, recorded.Body.String())
	}
	if asked != "/plugin/pages/2?path=Books%2Fa.pdf&wide=800" || auth != "Bearer secret" || cookie != "" {
		t.Errorf("the plugin was asked %q with %q and cookie %q", asked, auth, cookie)
	}
	if recorded.Header().Get("Set-Cookie") != "" {
		t.Error("the plugin set a cookie")
	}
	if policy := recorded.Header().Get("Content-Security-Policy"); policy != "" {
		t.Errorf("the plugin set the policy %q", policy)
	}
	if host.getReleased() != 1 {
		t.Errorf("the plugin was released %d times", host.getReleased())
	}
}

func TestServeRouteRejectsAPathOutsideTheRoute(t *testing.T) {
	reached := false
	plugin := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true }))
	t.Cleanup(plugin.Close)
	proxy := Open(newRouteHost(t, plugin))

	for _, path := range []string{
		"/plugin/org.example.reader/../secret",
		"/plugin/org.example.reader/pages/../../numen.v1.PluginLifecycleService/Shutdown",
		"/plugin/org.example.reader/./pages/1",
		"/plugin/org.example.reader//etc/passwd",
		`/plugin/org.example.reader/pages\..\x`,
		"/plugin/org.example.reader/",
		"/plugin/org.example.reader",
		"/plugin//pages/1",
	} {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		req.URL.Path = path
		recorded := httptest.NewRecorder()
		proxy.ServeRoute(recorded, req)
		if recorded.Code != http.StatusBadRequest {
			t.Errorf("%q was answered %d", path, recorded.Code)
		}
	}
	if reached {
		t.Error("a path outside the route reached the plugin")
	}
}

func TestServeRouteAnswersNotFoundForAPluginThatIsNotEnabled(t *testing.T) {
	recorded := httptest.NewRecorder()

	Open(&servingHost{}).ServeRoute(recorded, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/plugin/org.example.reader/pages/1", nil))

	if recorded.Code != http.StatusNotFound {
		t.Errorf("answered %d", recorded.Code)
	}
}

func TestServeRouteRejectsAMethodThatWrites(t *testing.T) {
	recorded := httptest.NewRecorder()

	Open(&servingHost{}).ServeRoute(recorded, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/plugin/org.example.reader/pages/1", nil))

	if recorded.Code != http.StatusMethodNotAllowed {
		t.Errorf("answered %d", recorded.Code)
	}
}
