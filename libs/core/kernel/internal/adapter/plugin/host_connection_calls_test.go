package plugin_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
)

// noModelCalls answers every call with an empty done payload.
type noModelCalls struct{}

func (noModelCalls) Call(context.Context, string, modelcall.Request, modelcall.Send) ([]byte, error) {
	return nil, nil
}

const providerManifest = `{
	"id": "md.example.provider", "name": "Provider", "version": "1.0.0", "targets": ["desktop.editor"],
	"services": ["numen.v1.ProviderService"], "backend": {"entrypoint": "bin/provider"},
	"contributes": {"connectionTypes": [{"id": "server", "title": "Server", "operations": ["llm.complete"],
		"fields": [{"key": "baseUrl", "type": "string", "title": "Address", "isAddress": true}]}]}
}`

func newProviderHost(t *testing.T) (*container.PluginRoles, func() []error, string) {
	t.Helper()
	dir := t.TempDir()
	host := newBuiltInHost(t, dir)
	var mu sync.Mutex
	var reported []error
	host.Failures.SetErrorHandler(func(err error) {
		mu.Lock()
		defer mu.Unlock()
		reported = append(reported, err)
	})
	registerBuiltIn(t, host, []byte(providerManifest), newBuiltInBackend().serve("built-in provider"))
	return host, func() []error {
		mu.Lock()
		defer mu.Unlock()
		return append([]error(nil), reported...)
	}, dir
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

	if err := host.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}

	found, ok := host.Registry.GetPlugin("md.example.provider")
	if !ok || found.Dir != "" {
		t.Fatalf("GetPlugin = %+v, %v, want the built-in provider", found, ok)
	}
	got := reported()
	if len(got) != 1 || !errors.Is(got[0], plugin.ErrProviderIDTaken) {
		t.Errorf("reported = %v, want the refusal of the plugin on disk", got)
	}
}

func TestAPluginOnDiskReplacesABuiltInThatProvidesNoConnectionType(t *testing.T) {
	host, _, dir := newProviderHost(t)
	simple := `{"id": "md.example.plain", "name": "Plain", "version": "1.0.0", "targets": ["desktop.editor"],
		"services": ["numen.v1.PluginToolService"], "backend": {"entrypoint": "bin/plain"}}`
	registerBuiltIn(t, host, []byte(simple), newBuiltInBackend().serve("built-in plain"))
	writeProviderOnDisk(t, dir, simple)

	if err := host.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}

	if found, ok := host.Registry.GetPlugin("md.example.plain"); !ok || found.Dir == "" {
		t.Errorf("GetPlugin = %+v, %v, want the plugin on disk", found, ok)
	}
}
