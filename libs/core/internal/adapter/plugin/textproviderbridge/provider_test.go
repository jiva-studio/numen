package textproviderbridge_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/textproviderbridge"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/textport"
)

// texts is a plugin's TextProviderService.
type texts struct {
	mu      sync.Mutex
	created []string
	auth    string
}

func (s *texts) CreateText(_ context.Context, req *connect.Request[v1.TextProviderServiceCreateTextRequest]) (*connect.Response[v1.TextProviderServiceCreateTextResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.auth = req.Header().Get("Authorization")
	s.created = append(s.created, req.Msg.GetPath()+"<"+req.Msg.GetSourceVersionId()+"/"+strconv.FormatBool(req.Msg.GetIsUserStarted()))
	return connect.NewResponse(&v1.TextProviderServiceCreateTextResponse{}), nil
}

// host knows three plugins: one that makes text, one that makes none, and one switched off.
type host struct {
	pluginport.PluginDiscoverer
	pluginport.PluginResolver
	endpoint pluginport.Endpoint
	released int
}

func (h *host) GetPluginInfos() []pluginport.PluginInfo {
	return []pluginport.PluginInfo{{ID: "org.example.off"}, {ID: "org.example.ocr"}, {ID: "org.example.other"}}
}

func (h *host) GetPlugin(id string) (pluginport.DiscoveredPlugin, bool) {
	switch id {
	case "org.example.ocr":
		return pluginport.DiscoveredPlugin{Manifest: pluginport.PluginManifest{ID: id, Name: "Example OCR", TextInput: &pluginport.PluginTextInput{
			ContentTypes: []domain.ContentType{"application/pdf"}, Kinds: []domain.TextKind{"article"}, IsAutoRun: true,
		}}}, true
	case "org.example.other":
		return pluginport.DiscoveredPlugin{Manifest: pluginport.PluginManifest{ID: id}}, true
	}
	return pluginport.DiscoveredPlugin{}, false
}

func (h *host) ListExtensions(mask domain.ContentType) []string {
	if mask == "application/pdf" {
		return []string{"pdf"}
	}
	return nil
}

func (h *host) EnsureActive(context.Context, string) (pluginport.Endpoint, error) {
	return h.endpoint, nil
}

func (h *host) Release(string) { h.released++ }

func newHost(t *testing.T) (*host, *texts) {
	t.Helper()
	served := &texts{}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewTextProviderServiceHandler(served))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	address, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(address.Port())
	if err != nil {
		t.Fatal(err)
	}
	return &host{endpoint: pluginport.Endpoint{Host: address.Hostname(), Port: port, Token: "secret"}}, served
}

func TestTheTextPluginsAreTheEnabledOnesThatDeclareAnInput(t *testing.T) {
	h, _ := newHost(t)
	got := textproviderbridge.NewPlugins(h).GetTextPlugins()
	if len(got) != 1 || got[0].ID != "org.example.ocr" || got[0].Name != "Example OCR" || !got[0].IsAutoRun || !got[0].Files.IsFileAccepted("a.pdf") {
		t.Fatalf("got %+v, want the OCR plugin alone", got)
	}
}

func TestAProviderCreatesTextThroughThePlugin(t *testing.T) {
	ctx := t.Context()
	h, served := newHost(t)
	provider, err := textproviderbridge.NewPlugins(h).OpenProvider(ctx, "org.example.ocr")
	if err != nil {
		t.Fatal(err)
	}
	if err := provider.CreateText(ctx, textport.TextCreation{Vault: "v1", Path: "a.pdf", Source: "x/y", IsUserStarted: true}); err != nil {
		t.Fatal(err)
	}
	if len(served.created) != 1 || served.created[0] != "a.pdf<x/y/true" {
		t.Fatalf("created %v", served.created)
	}
	if served.auth != "Bearer secret" {
		t.Fatalf("the call carried %q, want the plugin's token", served.auth)
	}
	if h.released != 1 {
		t.Fatalf("%d references released, want one for the call", h.released)
	}
}

func TestNoProviderIsOpenedForAPluginThatIsOffOrMakesNoText(t *testing.T) {
	h, _ := newHost(t)
	plugins := textproviderbridge.NewPlugins(h)
	var unavailable *pluginport.UnavailableError
	if _, err := plugins.OpenProvider(t.Context(), "org.example.off"); !errors.As(err, &unavailable) {
		t.Fatal("a disabled plugin was given a provider")
	}
	if _, err := plugins.OpenProvider(t.Context(), "org.example.other"); err == nil {
		t.Fatal("a plugin that makes no text was given a provider")
	}
}
