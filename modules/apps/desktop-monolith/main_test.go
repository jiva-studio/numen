package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// newLinkedConfig is a configuration whose plugin host serves every plugin this binary links in,
// and no plugin from disk.
func newLinkedConfig(t *testing.T) container.Config {
	t.Helper()
	host := container.NewPluginHost("desktop.editor")
	t.Cleanup(func() { _ = host.TeardownAll(context.WithoutCancel(t.Context()), time.Second) })
	registerPlugins(func(manifest []byte, serve serveFunc) {
		if err := host.RegisterBuiltIn(manifest, serve); err != nil {
			t.Fatalf("RegisterBuiltIn: %v", err)
		}
	})
	return container.Config{PluginHost: host}
}

// Each linked plugin is registered once, by the id its own manifest.json gives.
func TestEachPluginIsRegisteredOnceByItsOwnManifest(t *testing.T) {
	var ids []string
	registerPlugins(func(manifest []byte, serve serveFunc) {
		var declared struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(manifest, &declared); err != nil {
			t.Fatalf("a manifest does not parse: %v", err)
		}
		if serve == nil {
			t.Errorf("%s has no backend", declared.ID)
		}
		ids = append(ids, declared.ID)
	})

	slices.Sort(ids)
	if want := []string{
		"md.numen.agent", "md.numen.agent-claude", "md.numen.downloader", "md.numen.embeddings", "md.numen.epub",
		"md.numen.ocr", "md.numen.pdf", "md.numen.proofreading", "md.numen.transcription",
	}; !slices.Equal(ids, want) {
		t.Fatalf("registered %v, want %v", ids, want)
	}
}

// A document is read by the PDF reader linked into this binary, through the plugin host.
func TestADocumentIsReadByTheLinkedReader(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "plugins", "reader-pdf", "backend", "pdf", "testdata", "tiny.pdf"))
	if err != nil {
		t.Fatal(err)
	}

	layer, err := newLinkedConfig(t).TextExtractor().Read(t.Context(), raw)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(layer.Pages) == 0 || strings.TrimSpace(layer.Text) == "" {
		t.Fatalf("read %d pages and %q", len(layer.Pages), layer.Text)
	}
}

type heldKeys map[string]string

func (k heldKeys) GetKey(_ context.Context, name string) (string, error) {
	if key, ok := k[name]; ok {
		return key, nil
	}
	return "", port.ErrNoKey
}

func (k heldKeys) SetKey(context.Context, string, string) error { return nil }

func (k heldKeys) DeleteKey(context.Context, string) error { return nil }

// Numen Agent is configured with the base URL of its setting and the key the host holds, and asks
// the chat API they name.
func TestNumenAgentIsConfiguredWithItsBaseURLAndKey(t *testing.T) {
	var authorization string
	chat := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"id": "small-model"}}})
	}))
	defer chat.Close()
	host := newLinkedConfig(t).PluginHost
	host.SetKeyStore(heldKeys{"md.numen.agent/apiKey": "chat-key"})
	host.SetSettings(map[string]map[string]any{
		"md.numen.agent": {"baseUrl": chat.URL, port.SettingStoredSecrets: []any{"apiKey"}},
	})

	endpoint, err := host.EnsureActive(t.Context(), "md.numen.agent")
	if err != nil {
		t.Fatalf("EnsureActive: %v", err)
	}
	client := numenv1connect.NewAgentServiceClient(newClient(endpoint), endpoint.BaseURL(), withToken(endpoint.Token))
	res, err := client.GetAgentPrograms(t.Context(), connect.NewRequest(&numenv1.GetAgentProgramsRequest{}))
	if err != nil {
		t.Fatalf("GetAgentPrograms: %v", err)
	}

	if authorization != "Bearer chat-key" {
		t.Errorf("the chat API was asked with %q, want the key the host holds", authorization)
	}
	if programs := res.Msg.GetPrograms(); len(programs) != 1 || len(programs[0].GetModels()) != 1 {
		t.Errorf("the programs are %+v", programs)
	}
}
