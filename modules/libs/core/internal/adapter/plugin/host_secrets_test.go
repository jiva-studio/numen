package plugin

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

const (
	secretPluginID = "md.numen.agent"
	keychainValue  = "value-held-by-the-keychain"
	settingsValue  = "value-typed-into-the-settings"
)

// keys is a key store held in memory.
type keys struct {
	mu     sync.Mutex
	held   map[string]string
	failed error
}

func newKeys(held map[string]string) *keys {
	return &keys{held: held}
}

func (k *keys) GetKey(_ context.Context, name string) (string, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.failed != nil {
		return "", k.failed
	}
	key, found := k.held[name]
	if !found {
		return "", port.ErrNoKey
	}
	return key, nil
}

func (k *keys) SetKey(_ context.Context, name, key string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.failed != nil {
		return k.failed
	}
	k.held[name] = key
	return nil
}

func (k *keys) DeleteKey(_ context.Context, name string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if _, found := k.held[name]; !found {
		return port.ErrNoKey
	}
	delete(k.held, name)
	return nil
}

func (k *keys) hold(name string) (string, bool) {
	k.mu.Lock()
	defer k.mu.Unlock()
	key, found := k.held[name]
	return key, found
}

// configured records the Configure calls a plugin receives.
type configured struct {
	numenv1connect.UnimplementedPluginLifecycleServiceHandler
	mu    sync.Mutex
	calls []*v1.ConfigureRequest
}

func (c *configured) Configure(
	_ context.Context, req *connect.Request[v1.ConfigureRequest],
) (*connect.Response[v1.ConfigureResponse], error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls = append(c.calls, req.Msg)
	return connect.NewResponse(&v1.ConfigureResponse{Success: true}), nil
}

func (c *configured) last(t *testing.T) *v1.ConfigureRequest {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.calls) == 0 {
		t.Fatal("the plugin was not configured")
	}
	return c.calls[len(c.calls)-1]
}

// newSecretHost returns a host with a running agent plugin that declares the apiKey secret.
func newSecretHost(t *testing.T, store port.KeyStore) (*Host, *configured) {
	t.Helper()
	probe := &configured{}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(probe))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	host := NewHost("desktop.editor")
	t.Cleanup(func() { _ = host.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond) })
	host.plugins[secretPluginID] = DiscoveredPlugin{Manifest: port.PluginManifest{
		ID:      secretPluginID,
		Secrets: []string{"apiKey"},
		Settings: []Setting{
			{Key: "model", Type: "string", Title: "Model", Default: "sonnet"},
		},
	}}
	sup := NewSupervisor()
	sup.ActivateInProcess(time.Now())
	host.supervisors[secretPluginID] = sup
	host.endpoints[secretPluginID] = Endpoint{
		Host: "127.0.0.1", Port: server.Listener.Addr().(*net.TCPAddr).Port, Token: "launch-name",
	}
	if store != nil {
		host.SetKeyStore(store)
	}
	return host, probe
}

func assertNoKey(t *testing.T, where, text string, keys ...string) {
	t.Helper()
	for _, key := range keys {
		if strings.Contains(text, key) {
			t.Errorf("%s holds a key: %q", where, text)
		}
	}
}

// The key arrives as the value of its secret in Configure, and nowhere else in the call.
func TestTheKeyReachesThePluginOnlyAsTheValueOfItsSecret(t *testing.T) {
	store := newKeys(map[string]string{secretPluginID + "/apiKey": keychainValue})
	host, probe := newSecretHost(t, store)

	if err := host.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}
	call := probe.last(t)
	if call.GetValues()["apiKey"] != keychainValue {
		t.Errorf("values = %v, want the stored key under apiKey", call.GetValues())
	}
	assertNoKey(t, "the configuration json", call.GetConfigurationJson(), keychainValue)
	for name, value := range call.GetValues() {
		if name == "apiKey" {
			continue
		}
		assertNoKey(t, "value "+name, value, keychainValue)
	}
	if call.GetValues()["model"] != "sonnet" {
		t.Errorf("values = %v, want the model setting", call.GetValues())
	}
}

// A secret without a key is handed over empty, which tells the plugin it has none.
func TestASecretWithoutAKeyIsHandedOverEmpty(t *testing.T) {
	host, probe := newSecretHost(t, newKeys(map[string]string{}))
	if err := host.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}
	if got, found := probe.last(t).GetValues()["apiKey"]; !found || got != "" {
		t.Errorf("apiKey = %q, %v for a key nobody stored", got, found)
	}
}

// A key typed into the settings file under a secret's name is not a key: it
// reaches the plugin by none of the ways the host speaks to it.
func TestAKeyTypedIntoTheSettingsIsNotPassedOn(t *testing.T) {
	host, probe := newSecretHost(t, newKeys(map[string]string{}))
	host.SetSettings(map[string]map[string]any{secretPluginID: {"apiKey": settingsValue, "model": "opus"}})

	if err := host.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}
	call := probe.last(t)
	assertNoKey(t, "the configuration json", call.GetConfigurationJson(), settingsValue)
	for name, value := range call.GetValues() {
		assertNoKey(t, "value "+name, value, settingsValue)
	}
	if call.GetValues()["apiKey"] != "" {
		t.Errorf("apiKey = %q, want none", call.GetValues()["apiKey"])
	}
	if call.GetValues()["model"] != "opus" {
		t.Errorf("values = %v: the other settings are still passed", call.GetValues())
	}

	cfg, err := host.GetPluginConfigJSON(secretPluginID)
	if err != nil {
		t.Fatal(err)
	}
	assertNoKey(t, "NUMEN_PLUGIN_CONFIG", cfg, settingsValue, keychainValue)
	if _, found := host.GetPluginSettings(secretPluginID)["apiKey"]; found {
		t.Error("the settings the window reads hold a secret")
	}
}

// What the host puts in a plugin process's environment and on its command line
// holds no key, whether the key is in the store or was typed into the settings.
func TestNoKeyIsInTheEnvironmentOrOnTheCommandLine(t *testing.T) {
	store := newKeys(map[string]string{secretPluginID + "/apiKey": keychainValue})
	host, _ := newSecretHost(t, store)
	host.SetSettings(map[string]map[string]any{secretPluginID: {"apiKey": settingsValue}})

	configJSON, err := host.GetPluginConfigJSON(secretPluginID)
	if err != nil {
		t.Fatal(err)
	}
	const launchName = "launch-name"
	vars := getPluginVars(secretPluginID, launchEnvironment{
		vaultPath: "/vault", port: 4000, hostPort: 4001, token: launchName, configJSON: configJSON,
	})
	args := getPluginArgs(4000, launchName)
	for _, entry := range append(vars, args...) {
		assertNoKey(t, "launch entry "+entry, entry, keychainValue, settingsValue)
	}
	if len(vars) == 0 || len(args) == 0 {
		t.Fatal("nothing was read")
	}
}

// A key store that cannot be read is an error naming it, and the plugin is
// still configured with everything else.
func TestAKeyStoreThatCannotBeReadFailsLoudly(t *testing.T) {
	store := newKeys(map[string]string{})
	store.failed = port.ErrNoKeyStore
	host, probe := newSecretHost(t, store)

	err := host.NotifyPluginConfig(t.Context(), secretPluginID)
	if !errors.Is(err, port.ErrNoKeyStore) {
		t.Errorf("error = %v, want port.ErrNoKeyStore", err)
	}
	if probe.last(t).GetValues()["model"] != "sonnet" {
		t.Error("the other settings were not passed on")
	}
	if _, sent := probe.last(t).GetValues()["apiKey"]; sent {
		t.Error("a key that could not be read was sent, which would clear the one the plugin holds")
	}
}

// A host with no key store fails the same way for a plugin that declares a secret.
func TestAHostWithoutAKeyStoreFailsForAPluginThatDeclaresASecret(t *testing.T) {
	host, probe := newSecretHost(t, nil)
	if err := host.NotifyPluginConfig(t.Context(), secretPluginID); !errors.Is(err, port.ErrNoKeyStore) {
		t.Errorf("error = %v, want port.ErrNoKeyStore", err)
	}
	if probe.last(t).GetValues()["apiKey"] != "" {
		t.Error("a key came from nowhere")
	}
	if err := host.SetPluginSecret(t.Context(), secretPluginID, "apiKey", "value"); !errors.Is(err, port.ErrNoKeyStore) {
		t.Errorf("set: error = %v, want port.ErrNoKeyStore", err)
	}
}

// A key set through the host lands in the store under the plugin and the name,
// reaches the running plugin at once, and is gone from it once deleted.
func TestAKeySetThroughTheHostReachesTheRunningPlugin(t *testing.T) {
	store := newKeys(map[string]string{})
	host, probe := newSecretHost(t, store)

	if err := host.SetPluginSecret(t.Context(), secretPluginID, "apiKey", keychainValue); err != nil {
		t.Fatal(err)
	}
	if got, _ := store.hold(secretPluginID + "/apiKey"); got != keychainValue {
		t.Errorf("stored %q", got)
	}
	if probe.last(t).GetValues()["apiKey"] != keychainValue {
		t.Errorf("the plugin was handed %v", probe.last(t).GetValues())
	}

	if err := host.DeletePluginSecret(t.Context(), secretPluginID, "apiKey"); err != nil {
		t.Fatal(err)
	}
	if _, found := store.hold(secretPluginID + "/apiKey"); found {
		t.Error("the key was not deleted")
	}
	if probe.last(t).GetValues()["apiKey"] != "" {
		t.Errorf("the plugin still holds %v", probe.last(t).GetValues())
	}
}

// Only a secret the manifest declares can be stored, and only for a plugin the host knows.
func TestOnlyADeclaredSecretIsStored(t *testing.T) {
	store := newKeys(map[string]string{})
	host, _ := newSecretHost(t, store)

	if err := host.SetPluginSecret(t.Context(), secretPluginID, "other", "value"); !errors.Is(err, errUndeclaredSecret) {
		t.Errorf("an undeclared secret: error = %v", err)
	}
	if err := host.DeletePluginSecret(t.Context(), secretPluginID, "other"); !errors.Is(err, errUndeclaredSecret) {
		t.Errorf("an undeclared secret deleted: error = %v", err)
	}
	if err := host.SetPluginSecret(t.Context(), "md.numen.absent", "apiKey", "value"); !errors.Is(err, errPluginNotFound) {
		t.Errorf("an unknown plugin: error = %v", err)
	}
	if len(store.held) != 0 {
		t.Errorf("the store holds %v", store.held)
	}
}

// A store that fails to keep a key says so, and the plugin is not told it changed.
func TestAKeyTheStoreCannotKeepIsAnError(t *testing.T) {
	store := newKeys(map[string]string{})
	store.failed = port.ErrNoKeyStore
	host, probe := newSecretHost(t, store)

	if err := host.SetPluginSecret(t.Context(), secretPluginID, "apiKey", keychainValue); !errors.Is(err, port.ErrNoKeyStore) {
		t.Errorf("error = %v, want port.ErrNoKeyStore", err)
	}
	if len(probe.calls) != 0 {
		t.Error("the plugin was configured after a failed store")
	}
}
