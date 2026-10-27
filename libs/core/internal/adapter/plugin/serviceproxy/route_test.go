package serviceproxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// newRouteHost is a host running one plugin, org.example.reader, at the server given.
func newRouteHost(t *testing.T, plugin *httptest.Server, origins ...string) *servingHost {
	t.Helper()
	return &servingHost{
		endpoints: map[string]pluginport.Endpoint{"org.example.reader": endpointOf(t, plugin, "secret")},
		manifests: map[string]pluginport.PluginManifest{"org.example.reader": {ID: "org.example.reader", Origins: origins}},
	}
}

func TestServeRouteForwardsBelowThePluginRoute(t *testing.T) {
	var asked, auth, cookie string
	plugin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked, auth, cookie = r.URL.RequestURI(), r.Header.Get("Authorization"), r.Header.Get("Cookie")
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Content-Security-Policy", "default-src *")
		_, _ = io.WriteString(w, "page bytes")
	}))
	t.Cleanup(plugin.Close)
	host := newRouteHost(t, plugin, "https://tiles.example.org")
	req := httptest.NewRequest(http.MethodGet, "/plugin/org.example.reader/pages/2?path=Books%2Fa.pdf&wide=800", nil)
	req.Header.Set("Cookie", "session=1")
	recorded := httptest.NewRecorder()

	Open(host).ServeRoute(recorded, req)

	if recorded.Code != http.StatusOK || recorded.Body.String() != "page bytes" {
		t.Fatalf("answered %d %q", recorded.Code, recorded.Body.String())
	}
	if asked != "/plugin/pages/2?path=Books%2Fa.pdf&wide=800" || auth != "Bearer secret" || cookie != "" {
		t.Errorf("the plugin was asked %q with %q and cookie %q", asked, auth, cookie)
	}
	policy := recorded.Header().Get("Content-Security-Policy")
	if !strings.Contains(policy, "connect-src 'self' https://tiles.example.org;") || strings.Contains(policy, "default-src *") {
		t.Errorf("the policy reads %q", policy)
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
		req := httptest.NewRequest(http.MethodGet, "/", nil)
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

	Open(&servingHost{}).ServeRoute(recorded, httptest.NewRequest(http.MethodGet, "/plugin/org.example.reader/pages/1", nil))

	if recorded.Code != http.StatusNotFound {
		t.Errorf("answered %d", recorded.Code)
	}
}

func TestServeRouteRejectsAMethodThatWrites(t *testing.T) {
	recorded := httptest.NewRecorder()

	Open(&servingHost{}).ServeRoute(recorded, httptest.NewRequest(http.MethodPost, "/plugin/org.example.reader/pages/1", nil))

	if recorded.Code != http.StatusMethodNotAllowed {
		t.Errorf("answered %d", recorded.Code)
	}
}
