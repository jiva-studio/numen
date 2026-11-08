package plugin

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/port/connectionport"
)

type noModelCalls struct{}

func (noModelCalls) Complete(context.Context, string, connectionport.CompleteRequest, connectionport.Priority) (connectionport.CompleteResult, error) {
	return connectionport.CompleteResult{}, nil
}

func (noModelCalls) Embed(context.Context, string, []string, connectionport.Priority) ([][]float32, error) {
	return nil, nil
}

func (noModelCalls) GetEmbeddingProfile(context.Context, string) (connectionport.EmbeddingProfile, error) {
	return connectionport.EmbeddingProfile{}, nil
}

// The host serves its endpoint once something answers the calls on models, and gives it to the plugins.
func TestTheHostServesItsEndpointOnceItHasModelCalls(t *testing.T) {
	host := NewHost(testsupport.GetMachine(), "desktop.editor")
	if got, err := host.openHostEndpoint(t.Context()); err != nil || got != 0 {
		t.Fatalf("without model calls: port %d, %v", got, err)
	}
	if host.GetModelCalls() != nil {
		t.Fatal("a host with no model calls returned some")
	}

	host.SetModelCalls(noModelCalls{})

	if host.GetModelCalls() == nil {
		t.Error("the model calls set were not returned")
	}
	port, err := host.openHostEndpoint(t.Context())
	if err != nil || port <= 0 {
		t.Errorf("with model calls: port %d, %v", port, err)
	}
	if err := host.TeardownAll(t.Context(), time.Millisecond); err != nil {
		t.Fatal(err)
	}
}

const providerManifest = `{
	"id": "md.example.provider", "name": "Provider", "version": "1.0.0", "targets": ["desktop.editor"],
	"services": ["numen.v1.ProviderService"], "backend": {"entrypoint": "bin/provider"},
	"contributes": {"connectionTypes": [{"id": "server", "title": "Server", "operations": ["complete"],
		"fields": [{"key": "baseUrl", "type": "string", "title": "Address", "isAddress": true}]}]}
}`

func newProviderHost(t *testing.T) (*Host, *[]error, string) {
	t.Helper()
	tmp := t.TempDir()
	host := newBuiltInHost(t, tmp)
	var mu sync.Mutex
	var reported []error
	host.SetErrorHandler(func(err error) {
		mu.Lock()
		defer mu.Unlock()
		reported = append(reported, err)
	})
	registerBuiltIn(t, host, []byte(providerManifest), newBuiltInBackend().serve("built-in provider"))
	return host, &reported, tmp
}

func writeProviderOnDisk(t *testing.T, dir, manifest string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "provider"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "provider", "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The keys of a connection go to the plugin that provides its type, so a plugin found on disk under the
// id of a built-in provider is not replaced by it.
func TestAPluginOnDiskDoesNotReplaceABuiltInProvider(t *testing.T) {
	host, reported, dir := newProviderHost(t)
	writeProviderOnDisk(t, dir, providerManifest)
	host.dirs = []string{dir}

	if err := host.Discover(); err != nil {
		t.Fatal(err)
	}

	found, ok := host.GetPlugin("md.example.provider")
	if !ok || found.Dir != "" {
		t.Fatalf("GetPlugin = %+v, %v, want the built-in provider", found, ok)
	}
	if len(*reported) != 1 || !errors.Is((*reported)[0], errProviderIDTaken) {
		t.Errorf("reported = %v, want the refusal of the plugin on disk", *reported)
	}
}

func TestAPluginOnDiskReplacesABuiltInThatProvidesNoConnectionType(t *testing.T) {
	host, _, dir := newProviderHost(t)
	simple := `{"id": "md.example.plain", "name": "Plain", "version": "1.0.0", "targets": ["desktop.editor"],
		"services": ["numen.v1.AgentService"], "backend": {"entrypoint": "bin/plain"}}`
	registerBuiltIn(t, host, []byte(simple), newBuiltInBackend().serve("built-in plain"))
	writeProviderOnDisk(t, dir, simple)
	host.dirs = []string{dir}

	if err := host.Discover(); err != nil {
		t.Fatal(err)
	}

	if found, ok := host.GetPlugin("md.example.plain"); !ok || found.Dir == "" {
		t.Errorf("GetPlugin = %+v, %v, want the plugin on disk", found, ok)
	}
}
