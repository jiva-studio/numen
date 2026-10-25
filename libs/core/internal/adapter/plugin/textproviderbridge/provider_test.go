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

// texts is a plugin's TextProviderService over one text kept under one key.
type texts struct {
	mu      sync.Mutex
	created []string
	deleted []string
	auth    string
}

func (s *texts) GetText(_ context.Context, req *connect.Request[v1.TextProviderServiceGetTextRequest]) (*connect.Response[v1.TextProviderServiceGetTextResponse], error) {
	s.mu.Lock()
	s.auth = req.Header().Get("Authorization")
	s.mu.Unlock()
	if req.Msg.GetKey() != "k" || req.Msg.GetVaultId() != "v1" {
		return connect.NewResponse(&v1.TextProviderServiceGetTextResponse{Error: v1.TextErrorCode_TEXT_ERROR_CODE_MISSING}), nil
	}
	if req.Msg.GetCursor() == "" {
		return connect.NewResponse(&v1.TextProviderServiceGetTextResponse{
			Parts:      []*v1.TextPart{{Text: "one ", Section: "Intro", Place: &v1.TextPlace{Page: 1, Rects: []*v1.TextRect{{MaxX: 1}}}}},
			NextCursor: "2",
		}), nil
	}
	return connect.NewResponse(&v1.TextProviderServiceGetTextResponse{Parts: []*v1.TextPart{{Text: "two"}}}), nil
}

func (s *texts) CreateText(_ context.Context, req *connect.Request[v1.TextProviderServiceCreateTextRequest]) (*connect.Response[v1.TextProviderServiceCreateTextResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.created = append(s.created, req.Msg.GetPath()+"<"+req.Msg.GetInputVersionId()+"/"+strconv.FormatBool(req.Msg.GetIsUserStarted()))
	return connect.NewResponse(&v1.TextProviderServiceCreateTextResponse{}), nil
}

func (s *texts) DeleteText(_ context.Context, req *connect.Request[v1.TextProviderServiceDeleteTextRequest]) (*connect.Response[v1.TextProviderServiceDeleteTextResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleted = append(s.deleted, req.Msg.GetKey())
	return connect.NewResponse(&v1.TextProviderServiceDeleteTextResponse{}), nil
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
			Files: []string{".pdf"}, Formats: []domain.TextFormat{domain.TextFormatPlain}, IsAutoRun: true,
		}}}, true
	case "org.example.other":
		return pluginport.DiscoveredPlugin{Manifest: pluginport.PluginManifest{ID: id}}, true
	}
	return pluginport.DiscoveredPlugin{}, false
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

func TestAProviderReadsCreatesAndDeletesThroughThePlugin(t *testing.T) {
	ctx := t.Context()
	h, served := newHost(t)
	plugins := textproviderbridge.NewPlugins(h)
	provider, err := plugins.OpenProvider(ctx, "org.example.ocr")
	if err != nil {
		t.Fatal(err)
	}
	first, err := provider.GetText(ctx, textport.TextRequest{Vault: "v1", Path: "a.pdf", Key: "k"})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Parts) != 1 || first.Parts[0].Section != "Intro" || first.Parts[0].Place.Page != 1 || first.NextCursor != "2" {
		t.Fatalf("first page %+v", first)
	}
	last, err := provider.GetText(ctx, textport.TextRequest{Vault: "v1", Path: "a.pdf", Key: "k", Cursor: first.NextCursor})
	if err != nil || last.NextCursor != "" || last.Parts[0].Text != "two" {
		t.Fatalf("last page %+v (%v)", last, err)
	}
	if served.auth != "Bearer secret" {
		t.Fatalf("the call carried %q, want the plugin's token", served.auth)
	}
	if _, err := provider.GetText(ctx, textport.TextRequest{Vault: "v1", Path: "a.pdf", Key: "gone"}); !errors.Is(err, textport.ErrNoText) {
		t.Fatalf("a missing text: got %v, want ErrNoText", err)
	}
	if err := provider.CreateText(ctx, textport.TextCreation{Vault: "v1", Path: "a.pdf", Input: "x/y", IsUserStarted: true}); err != nil {
		t.Fatal(err)
	}
	if err := provider.DeleteText(ctx, "v1", "a.pdf", "k"); err != nil {
		t.Fatal(err)
	}
	if len(served.created) != 1 || served.created[0] != "a.pdf<x/y/true" || len(served.deleted) != 1 {
		t.Fatalf("created %v, deleted %v", served.created, served.deleted)
	}
	if h.released != 5 {
		t.Fatalf("%d references released, want one for each of the 5 calls", h.released)
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
