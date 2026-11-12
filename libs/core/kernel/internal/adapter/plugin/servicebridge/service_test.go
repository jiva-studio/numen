package servicebridge

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jiva-studio/numen/libs/note/domain"
	catalogmanifest "github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
)

// servingHost serves the services it is given from the plugins it runs.
type servingHost struct {
	infos     []plugindomain.Info
	services  map[string]string
	endpoints map[string]transport.Endpoint
	manifests map[string]catalogmanifest.Manifest
	mu        sync.Mutex
	released  int
}

func (h *servingHost) ResolveCapability(string) (catalogmanifest.Discovered, error) {
	return catalogmanifest.Discovered{}, &plugindomain.UnavailableError{Reason: plugindomain.NotInstalled}
}

func (h *servingHost) GetPlugin(id string) (catalogmanifest.Discovered, bool) {
	manifest, found := h.manifests[id]
	return catalogmanifest.Discovered{Manifest: manifest}, found
}

func (h *servingHost) ResolveService(service string) (catalogmanifest.Discovered, bool) {
	id, serves := h.services[service]
	return catalogmanifest.Discovered{Manifest: catalogmanifest.Manifest{ID: id}}, serves
}

func (h *servingHost) EnsureActive(_ context.Context, id string) (transport.Endpoint, error) {
	endpoint, running := h.endpoints[id]
	if !running {
		return transport.Endpoint{}, errors.New("the plugin is down")
	}
	return endpoint, nil
}

func (h *servingHost) Release(string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.released++
}

func (h *servingHost) getReleasedCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.released
}

func endpointOf(t *testing.T, server *httptest.Server, token string) transport.Endpoint {
	t.Helper()
	host, rawPort, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	number, err := strconv.Atoi(rawPort)
	if err != nil {
		t.Fatal(err)
	}
	return transport.Endpoint{Host: host, Port: number, Token: token}
}

func TestProxyReachesPluginServingService(t *testing.T) {
	var path, query, method, auth, origin, body string
	plugin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, query, method, auth, origin = r.URL.Path, r.URL.RawQuery, r.Method, r.Header.Get("Authorization"), r.Header.Get("Origin")
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		body = string(raw)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"ok":true}`)); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(plugin.Close)
	host := &servingHost{
		services:  map[string]string{"numen.v1.SampleService": "md.numen.a"},
		endpoints: map[string]transport.Endpoint{"md.numen.a": endpointOf(t, plugin, "secret")},
	}
	window := httptest.NewServer(Open(host))
	t.Cleanup(window.Close)
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, window.URL+"/numen.v1.SampleService/ReadSample?x=1", strings.NewReader(`{"path":"a.md"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer from-the-window")
	req.Header.Set("Origin", "wails://wails")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	answer, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if err := res.Body.Close(); err != nil {
		t.Error(err)
	}

	if path != "/numen.v1.SampleService/ReadSample" || query != "x=1" || method != http.MethodPost || body != `{"path":"a.md"}` {
		t.Errorf("the plugin was asked %s %s?%s with %q", method, path, query, body)
	}
	if auth != "Bearer secret" || origin != "" {
		t.Errorf("the plugin was shown the authorization %q and the origin %q", auth, origin)
	}
	if string(answer) != `{"ok":true}` || res.Header.Get("Content-Type") != "application/json" {
		t.Errorf("the window was answered %q as %q", answer, res.Header.Get("Content-Type"))
	}
	window.Close()
	if host.getReleasedCount() != 1 {
		t.Errorf("the plugin was released %d times", host.getReleasedCount())
	}
}

func TestProxyStreamsToWindowMessageByMessage(t *testing.T) {
	second := make(chan struct{})
	plugin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte("one")); err != nil {
			t.Error(err)
		}
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Error(err)
		}
		<-second
		if _, err := w.Write([]byte("two")); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(plugin.Close)
	host := &servingHost{
		services:  map[string]string{"numen.v1.EventsService": "md.numen.a"},
		endpoints: map[string]transport.Endpoint{"md.numen.a": endpointOf(t, plugin, "secret")},
	}
	window := httptest.NewServer(Open(host))
	t.Cleanup(window.Close)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, window.URL+"/numen.v1.EventsService/WatchReloads", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/connect+json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	first := make([]byte, 3)
	_, readErr := io.ReadFull(res.Body, first)
	close(second)
	rest, restErr := io.ReadAll(res.Body)
	if restErr != nil {
		t.Error(restErr)
	}

	if readErr != nil || string(first) != "one" || string(rest) != "two" {
		t.Errorf("the window read %q (%v) and then %q", first, readErr, rest)
	}
}

func TestOnlyAServiceAPluginServesIsForwarded(t *testing.T) {
	host := &servingHost{services: map[string]string{"numen.v1.SampleService": "md.numen.a"}}
	proxy := Open(host)

	if !proxy.Serves("/numen.v1.SampleService/ReadSample") || proxy.Serves("/numen.v1.NoteService/Read") || proxy.Serves("/") {
		t.Error("the proxy claimed a service it does not forward, or rejected one it does")
	}
	recorded := httptest.NewRecorder()
	proxy.ServeHTTP(recorded, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/numen.v1.NoteService/Read", nil))
	if recorded.Code != http.StatusNotFound {
		t.Errorf("a service no plugin serves was answered %d", recorded.Code)
	}
}

func TestProxyUnavailableWhenPluginCannotStart(t *testing.T) {
	host := &servingHost{services: map[string]string{"numen.v1.SampleService": "md.numen.a"}}
	recorded := httptest.NewRecorder()

	Open(host).ServeHTTP(recorded, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/numen.v1.SampleService/ReadSample", nil))

	var response map[string]string
	if err := json.Unmarshal(recorded.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if recorded.Code != http.StatusServiceUnavailable || response["code"] != "unavailable" {
		t.Errorf("the answer was %d %v", recorded.Code, response)
	}
	if host.getReleasedCount() != 0 {
		t.Errorf("a plugin that never started was released %d times", host.getReleasedCount())
	}
}

func (h *servingHost) ResolveNoteProfile(domain.NoteType) (domain.NoteProfile, bool) {
	return domain.NoteProfile{}, false
}

func (h *servingHost) GetPluginInfos() []plugindomain.Info { return h.infos }
