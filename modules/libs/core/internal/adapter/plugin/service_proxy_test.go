package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port/pluginport"
)

// servingHost serves the services it is given from the plugins it runs.
type servingHost struct {
	services  map[string]string
	endpoints map[string]pluginport.Endpoint
	mu        sync.Mutex
	released  int
}

func (h *servingHost) ResolveCapability(string) (pluginport.DiscoveredPlugin, error) {
	return pluginport.DiscoveredPlugin{}, &pluginport.UnavailableError{Reason: pluginport.PluginNotInstalled}
}

func (h *servingHost) ResolveCapabilities(string) []pluginport.DiscoveredPlugin { return nil }

func (h *servingHost) GetPlugin(string) (pluginport.DiscoveredPlugin, bool) {
	return pluginport.DiscoveredPlugin{}, false
}

func (h *servingHost) ResolveService(service string) (pluginport.DiscoveredPlugin, bool) {
	id, serves := h.services[service]
	return pluginport.DiscoveredPlugin{Manifest: pluginport.PluginManifest{ID: id}}, serves
}

func (h *servingHost) EnsureActive(_ context.Context, id string) (pluginport.Endpoint, error) {
	endpoint, running := h.endpoints[id]
	if !running {
		return pluginport.Endpoint{}, errors.New("the plugin is down")
	}
	return endpoint, nil
}

func (h *servingHost) Release(string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.released++
}

func (h *servingHost) getReleased() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.released
}

func endpointOf(t *testing.T, server *httptest.Server, token string) pluginport.Endpoint {
	t.Helper()
	host, rawPort, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	number, err := strconv.Atoi(rawPort)
	if err != nil {
		t.Fatal(err)
	}
	return pluginport.Endpoint{Host: host, Port: number, Token: token}
}

func TestACallReachesThePluginThatServesItsService(t *testing.T) {
	var path, query, method, auth, origin, body string
	plugin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, query, method, auth, origin = r.URL.Path, r.URL.RawQuery, r.Method, r.Header.Get("Authorization"), r.Header.Get("Origin")
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(plugin.Close)
	host := &servingHost{
		services:  map[string]string{"numen.v1.CardsService": "md.numen.a"},
		endpoints: map[string]pluginport.Endpoint{"md.numen.a": endpointOf(t, plugin, "secret")},
	}
	window := httptest.NewServer(OpenServiceProxy(host))
	t.Cleanup(window.Close)
	req, _ := http.NewRequest(http.MethodPost, window.URL+"/numen.v1.CardsService/ReadDeck?x=1", strings.NewReader(`{"path":"a.md"}`))
	req.Header.Set("Authorization", "Bearer from-the-window")
	req.Header.Set("Origin", "wails://wails")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	answer, _ := io.ReadAll(res.Body)
	res.Body.Close()

	if path != "/numen.v1.CardsService/ReadDeck" || query != "x=1" || method != http.MethodPost || body != `{"path":"a.md"}` {
		t.Errorf("the plugin was asked %s %s?%s with %q", method, path, query, body)
	}
	if auth != "Bearer secret" || origin != "" {
		t.Errorf("the plugin was shown the authorization %q and the origin %q", auth, origin)
	}
	if string(answer) != `{"ok":true}` || res.Header.Get("Content-Type") != "application/json" {
		t.Errorf("the window was answered %q as %q", answer, res.Header.Get("Content-Type"))
	}
	window.Close()
	if host.getReleased() != 1 {
		t.Errorf("the plugin was released %d times", host.getReleased())
	}
}

func TestAStreamReachesTheWindowMessageByMessage(t *testing.T) {
	second := make(chan struct{})
	plugin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("one"))
		http.NewResponseController(w).Flush()
		<-second
		_, _ = w.Write([]byte("two"))
	}))
	t.Cleanup(plugin.Close)
	host := &servingHost{
		services:  map[string]string{"numen.v1.FlashcardsService": "md.numen.a"},
		endpoints: map[string]pluginport.Endpoint{"md.numen.a": endpointOf(t, plugin, "secret")},
	}
	window := httptest.NewServer(OpenServiceProxy(host))
	t.Cleanup(window.Close)

	res, err := http.Post(window.URL+"/numen.v1.FlashcardsService/WatchReloads", "application/connect+json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	first := make([]byte, 3)
	_, readErr := io.ReadFull(res.Body, first)
	close(second)
	rest, _ := io.ReadAll(res.Body)

	if readErr != nil || string(first) != "one" || string(rest) != "two" {
		t.Errorf("the window read %q (%v) and then %q", first, readErr, rest)
	}
}

func TestOnlyAServiceAPluginServesIsForwarded(t *testing.T) {
	host := &servingHost{services: map[string]string{"numen.v1.CardsService": "md.numen.a"}}
	proxy := OpenServiceProxy(host)

	if !proxy.Serves("/numen.v1.CardsService/ReadDeck") || proxy.Serves("/numen.v1.NoteService/Read") || proxy.Serves("/") {
		t.Error("the proxy claimed a service it does not forward, or rejected one it does")
	}
	recorded := httptest.NewRecorder()
	proxy.ServeHTTP(recorded, httptest.NewRequest(http.MethodPost, "/numen.v1.NoteService/Read", nil))
	if recorded.Code != http.StatusNotFound {
		t.Errorf("a service no plugin serves was answered %d", recorded.Code)
	}
}

func TestAPluginThatCannotBeStartedIsAnUnavailableService(t *testing.T) {
	host := &servingHost{services: map[string]string{"numen.v1.CardsService": "md.numen.a"}}
	recorded := httptest.NewRecorder()

	OpenServiceProxy(host).ServeHTTP(recorded, httptest.NewRequest(http.MethodPost, "/numen.v1.CardsService/ReadDeck", nil))

	var response map[string]string
	if err := json.Unmarshal(recorded.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if recorded.Code != http.StatusServiceUnavailable || response["code"] != "unavailable" {
		t.Errorf("the answer was %d %v", recorded.Code, response)
	}
	if host.getReleased() != 0 {
		t.Errorf("a plugin that never started was released %d times", host.getReleased())
	}
}

func (h *servingHost) ResolveNoteProfile(domain.NoteType) (domain.NoteProfile, bool) {
	return domain.NoteProfile{}, false
}
