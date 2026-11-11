package editor_test

import (
	"context"
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	windowsettings "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/servetest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/plugintest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const (
	alphaID = "md.numen.test.alpha"
	betaID  = "md.numen.test.beta"
	gammaID = "md.numen.test.gamma"
)

// Alpha declares the key for two providers (one with a group, one without) and a second secret with no group.
const alphaManifest = `{
	"id": "md.numen.test.alpha", "name": "Alpha", "version": "1.0.0", "targets": ["desktop.editor"],
	"backend": {"entrypoint": "bin/alpha"},
	"settings": [{"key": "provider", "type": "enum", "title": "Provider", "default": "openrouter",
		"options": ["openrouter", "openai", "custom"]}],
	"secrets": [
		{"name": "apiKey", "title": "API key", "description": "Key of the service", "group": "openrouter",
			"when": {"setting": "provider", "is": ["openrouter", "openai"]}},
		{"name": "apiKey", "title": "API key", "description": "Key of the custom service",
			"when": {"setting": "provider", "is": ["custom"]}},
		{"name": "orgId", "title": "Organization", "description": "Organization of the account"}
	]
}`

// Beta asks for the group openai.
const betaManifest = `{
	"id": "md.numen.test.beta", "name": "Beta", "version": "1.0.0", "targets": ["desktop.editor"],
	"backend": {"entrypoint": "bin/beta"},
	"secrets": [{"name": "apiKey", "title": "API key", "group": "openai"}]
}`

// Gamma asks for the group mistral.
const gammaManifest = `{
	"id": "md.numen.test.gamma", "name": "Gamma", "version": "1.0.0", "targets": ["desktop.editor"],
	"backend": {"entrypoint": "bin/gamma"},
	"secrets": [{"name": "apiKey", "title": "API key", "group": "mistral"}]
}`

// secretKeys is a key store held in memory.
type secretKeys struct {
	mu      sync.Mutex
	items   map[string]string
	failSet error
	reads   int
}

func (k *secretKeys) GetKey(_ context.Context, name string) (string, error) {
	k.mu.Lock()
	k.reads++
	k.mu.Unlock()
	k.mu.Lock()
	defer k.mu.Unlock()
	if value, found := k.items[name]; found {
		return value, nil
	}
	return "", plugindomain.ErrNoKey
}

func (k *secretKeys) SetKey(_ context.Context, name, value string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.failSet != nil {
		return k.failSet
	}
	k.items[name] = value
	return nil
}

func (k *secretKeys) DeleteKey(_ context.Context, name string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if _, found := k.items[name]; !found {
		return plugindomain.ErrNoKey
	}
	delete(k.items, name)
	return nil
}

func (k *secretKeys) snapshot() map[string]string {
	k.mu.Lock()
	defer k.mu.Unlock()
	copied := make(map[string]string, len(k.items))
	maps.Copy(copied, k.items)
	return copied
}

// probe is a plugin that records the Configure calls it receives.
type probe struct {
	servetest.PluginLifecycle
	mu       sync.Mutex
	calls    []*v1.ConfigureRequest
	failure  error
	hold     chan struct{}
	entered  chan struct{}
	attempts int
}

func (p *probe) Configure(
	ctx context.Context, req *connect.Request[v1.ConfigureRequest],
) (*connect.Response[v1.ConfigureResponse], error) {
	p.mu.Lock()
	p.attempts++
	hold, entered := p.hold, p.entered
	p.mu.Unlock()
	if entered != nil {
		select {
		case entered <- struct{}{}:
		default:
		}
	}
	if hold != nil {
		select {
		case <-hold:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.failure != nil {
		return nil, p.failure
	}
	p.calls = append(p.calls, req.Msg)
	return connect.NewResponse(&v1.ConfigureResponse{Success: true}), nil
}

func (p *probe) setFailure(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.failure = err
}

func (p *probe) attemptCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.attempts
}

func (p *probe) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.calls)
}

func (p *probe) last(t *testing.T) *v1.ConfigureRequest {
	t.Helper()
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.calls) == 0 {
		t.Fatal("the plugin was not configured")
	}
	return p.calls[len(p.calls)-1]
}

func (p *probe) all() []*v1.ConfigureRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]*v1.ConfigureRequest(nil), p.calls...)
}

func (p *probe) entry(t *testing.T, name string) (*v1.ConfigureSecrets, bool) {
	t.Helper()
	var found []*v1.ConfigureSecrets
	for _, one := range p.last(t).GetSecrets() {
		if one.GetName() == name {
			found = append(found, one)
		}
	}
	if len(found) > 1 {
		t.Fatalf("the call carries %d entries for %s, want at most one", len(found), name)
	}
	if len(found) == 0 {
		return nil, false
	}
	return found[0], true
}

func (p *probe) ids(t *testing.T, name string) []string {
	t.Helper()
	entry, _ := p.entry(t, name)
	var ids []string
	for _, value := range entry.GetValues() {
		ids = append(ids, value.GetId())
	}
	return ids
}

func (p *probe) serialized() string {
	var text strings.Builder
	for _, call := range p.all() {
		text.WriteString(protojson.Format(call))
	}
	return text.String()
}

// window is an installation with three plugins, a key store in memory and a client of the settings service.
type window struct {
	client   numenv1connect.SettingsServiceClient
	host     *plugin.Host
	keys     *secretKeys
	probes   map[string]*probe
	file     string
	pluginsd string
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func openSecretsWindow(t *testing.T, fileBody string) *window {
	t.Helper()
	pluginsDir := t.TempDir()
	writeFile(t, filepath.Join(pluginsDir, "alpha", "manifest.json"), alphaManifest)
	writeFile(t, filepath.Join(pluginsDir, "beta", "manifest.json"), betaManifest)
	writeFile(t, filepath.Join(pluginsDir, "gamma", "manifest.json"), gammaManifest)

	host := plugin.NewHost(testsupport.GetMachine(), "desktop.editor", pluginsDir)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := host.Teardown.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond); err != nil {
			t.Error(err)
		}
	})

	probes := map[string]*probe{alphaID: {}, betaID: {}, gammaID: {}}
	ports := map[string]int{}
	processes := map[string]*os.Process{}
	for id, one := range probes {
		processes[id] = plugintest.StartSleepProcess(t).Process
		mux := http.NewServeMux()
		mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(one))
		server := httptest.NewServer(mux)
		t.Cleanup(server.Close)
		ports[id] = plugintest.GetPort(t, server)
	}
	host.Processes.SetLauncher(func(_ context.Context, p manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		return processes[p.Manifest.ID], plugintest.GetReadyOutput(ports[p.Manifest.ID]), nil
	})

	configDir := t.TempDir()
	file := filepath.Join(configDir, "numen.json")
	writeFile(t, file, fileBody)
	keys := &secretKeys{items: map[string]string{}}
	cfg := container.NewConfig(container.Config{
		PluginHost: container.AdaptPluginHost(host),
		KeyStore:   keys,
		Paths:      container.Paths{SettingsPath: file, IndexPath: filepath.Join(t.TempDir(), "index.db"), RegistryPath: filepath.Join(configDir, "vaults.json"), PluginCachePath: filepath.Join(t.TempDir(), "plugins")},
	})
	cfg = applySettings(t, cfg)

	root := t.TempDir()
	if _, err := vaultconfig.Initialize(root, vaultconfig.DefaultServiceDir, testsupport.GetEpoch()); err != nil {
		t.Fatal(err)
	}
	if _, err := cfg.AddVault(root, "secrets"); err != nil {
		t.Fatal(err)
	}
	installation, _, err := openRegistered(cfg, t.Context(), "", os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, installation)

	path, handler := numenv1connect.NewSettingsServiceHandler(editor.HandlerOf[*windowsettings.Handler](installation))
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.CloseClientConnections)
	t.Cleanup(server.Close)

	w := &window{
		client:   numenv1connect.NewSettingsServiceClient(server.Client(), server.URL),
		host:     host,
		keys:     keys,
		probes:   probes,
		file:     file,
		pluginsd: pluginsDir,
	}
	for _, id := range []string{alphaID, betaID, gammaID} {
		if _, err := host.Activation.EnsureActive(t.Context(), id); err != nil {
			t.Fatal(err)
		}
		plugintest.WaitUntil(t, id+" is configured", func() bool { return probes[id].count() > 0 })
	}
	return w
}

func (w *window) create(t *testing.T, group, name, value string) string {
	t.Helper()
	response, err := w.client.CreateSecret(t.Context(), connect.NewRequest(&v1.CreateSecretRequest{Group: group, Name: name, Value: value}))
	if err != nil {
		t.Fatalf("CreateSecret %s/%s: %v", group, name, err)
	}
	return response.Msg.GetId()
}

func (w *window) allow(t *testing.T, secretID, pluginID, name, group string) {
	t.Helper()
	_, err := w.client.AllowPluginSecret(t.Context(), connect.NewRequest(&v1.AllowPluginSecretRequest{
		SecretId: secretID, PluginId: pluginID, Name: name, Group: group,
	}))
	if err != nil {
		t.Fatalf("AllowPluginSecret %s to %s/%s/%s: %v", secretID, pluginID, name, group, err)
	}
}

func (w *window) deny(t *testing.T, secretID, pluginID, name, group string) {
	t.Helper()
	_, err := w.client.DenyPluginSecret(t.Context(), connect.NewRequest(&v1.DenyPluginSecretRequest{
		SecretId: secretID, PluginId: pluginID, Name: name, Group: group,
	}))
	if err != nil {
		t.Fatalf("DenyPluginSecret: %v", err)
	}
}

func (w *window) get(t *testing.T) *v1.GetSettingsResponse {
	t.Helper()
	response, err := w.client.GetSettings(t.Context(), connect.NewRequest(&v1.GetSettingsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	return response.Msg
}

func (w *window) secrets(t *testing.T) []*v1.Secret {
	t.Helper()
	var secrets []*v1.Secret
	for _, section := range w.get(t).GetSecretSections() {
		secrets = append(secrets, section.GetSecrets()...)
	}
	return secrets
}

func (w *window) hasValue(t *testing.T, id string) bool {
	t.Helper()
	response, err := w.client.ListSecretStates(t.Context(), connect.NewRequest(&v1.ListSecretStatesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range response.Msg.GetStates() {
		if one.GetId() == id {
			return one.GetHasValue()
		}
	}
	t.Fatalf("no state for %s", id)
	return false
}

func (w *window) find(t *testing.T, id string) *v1.Secret {
	t.Helper()
	for _, one := range w.secrets(t) {
		if one.GetId() == id {
			return one
		}
	}
	return nil
}

func (w *window) setProvider(t *testing.T, value string) {
	t.Helper()
	_, err := w.client.WriteSettings(t.Context(), connect.NewRequest(&v1.WriteSettingsRequest{
		Settings: []*v1.Setting{{At: []string{alphaID, "provider"}, Value: `"` + value + `"`}},
	}))
	if err != nil {
		t.Fatal(err)
	}
}

func (w *window) readFile(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(w.file)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func (w *window) fileSecrets(t *testing.T) []map[string]any {
	t.Helper()
	var root struct {
		Secrets []map[string]any `json:"secrets"`
	}
	if err := json.Unmarshal([]byte(w.readFile(t)), &root); err != nil {
		t.Fatal(err)
	}
	return root.Secrets
}

func ids(secrets []*v1.Secret) []string {
	var list []string
	for _, one := range secrets {
		list = append(list, one.GetId())
	}
	return list
}

func applySettings(t *testing.T, cfg container.Config) container.Config {
	t.Helper()
	cfg, _, err := cfg.ApplySettings()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { container.UnsubscribeHost(cfg) })
	return cfg
}
