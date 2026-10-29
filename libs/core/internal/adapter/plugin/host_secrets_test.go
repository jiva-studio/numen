package plugin

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/manifestfile"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/process"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

const (
	secretPluginID = "md.numen.agents.numen"
	declared       = "apiKey"
	fromKeychain   = "value-held-by-the-keychain"
	fromSettings   = "value-typed-into-the-settings"
)

// keys is a key store held in memory.
type keys struct {
	mu      sync.Mutex
	item    map[string]string
	failed  error
	entered chan struct{}
	gate    chan struct{}
}

func newKeys(target map[string]string) *keys {
	return &keys{item: target}
}

func (k *keys) GetKey(_ context.Context, name string) (string, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.failed != nil {
		return "", k.failed
	}
	key, found := k.item[name]
	if !found {
		return "", pluginport.ErrNoKey
	}
	return key, nil
}

func (k *keys) SetKey(_ context.Context, name, key string) error {
	if k.gate != nil {
		k.entered <- struct{}{}
		<-k.gate
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.failed != nil {
		return k.failed
	}
	k.item[name] = key
	return nil
}

func (k *keys) DeleteKey(_ context.Context, name string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if _, found := k.item[name]; !found {
		return pluginport.ErrNoKey
	}
	delete(k.item, name)
	return nil
}

func (k *keys) hold(name string) (string, bool) {
	k.mu.Lock()
	defer k.mu.Unlock()
	key, found := k.item[name]
	return key, found
}

// configured records the Configure calls a plugin receives.
type configured struct {
	testsupport.PluginLifecycle
	mu            sync.Mutex
	calls         []*v1.ConfigureRequest
	unimplemented bool
	shutdowns     int
}

func (c *configured) Configure(
	_ context.Context, req *connect.Request[v1.ConfigureRequest],
) (*connect.Response[v1.ConfigureResponse], error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls = append(c.calls, req.Msg)
	if c.unimplemented {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("no configure"))
	}
	return connect.NewResponse(&v1.ConfigureResponse{Success: true}), nil
}

// Shutdown counts the shutdown requests the plugin receives.
func (c *configured) Shutdown(
	_ context.Context, _ *connect.Request[v1.ShutdownRequest],
) (*connect.Response[v1.ShutdownResponse], error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.shutdowns++
	return connect.NewResponse(&v1.ShutdownResponse{Success: true}), nil
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
func newSecretHost(t *testing.T, store pluginport.KeyStore) (*Host, *configured) {
	t.Helper()
	probe := &configured{}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(probe))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	host := NewHost("desktop.editor")
	t.Cleanup(func() { _ = host.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond) })
	host.plugins[secretPluginID] = manifestfile.DiscoveredPlugin{Manifest: pluginport.PluginManifest{
		ID:      secretPluginID,
		Secrets: []pluginport.PluginSecret{{Name: "apiKey", Title: "API key"}},
		Settings: []manifestfile.Setting{
			{Key: "model", Type: "string", Title: "Model", Default: "sonnet"},
		},
	}}
	sup := lifecycle.New()
	sup.ActivateInProcess(time.Now())
	host.supervisors[secretPluginID] = sup
	host.endpoints[secretPluginID] = transport.Endpoint{
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
	store := newKeys(map[string]string{secretPluginID + "/apiKey": fromKeychain})
	host, probe := newSecretHost(t, store)
	recordStored(host)

	if err := host.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}
	call := probe.last(t)
	if call.GetValues()["apiKey"] != fromKeychain {
		t.Errorf("values = %v, want the stored key under apiKey", call.GetValues())
	}
	assertNoKey(t, "the configuration json", call.GetConfigurationJson(), fromKeychain)
	for name, value := range call.GetValues() {
		if name == "apiKey" {
			continue
		}
		assertNoKey(t, "value "+name, value, fromKeychain)
	}
	if call.GetValues()["model"] != "sonnet" {
		t.Errorf("values = %v, want the model setting", call.GetValues())
	}
}

// A secret without a key is delivered as empty, which tells the plugin it has
// no value.
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
func TestAKeyTypedIntoTheSettingsIsNotPassedOn(t *testing.T) {
	host, probe := newSecretHost(t, newKeys(map[string]string{}))
	host.SetSettings(map[string]map[string]any{secretPluginID: {declared: fromSettings, "model": "opus"}})

	if err := host.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}
	call := probe.last(t)
	assertNoKey(t, "the configuration json", call.GetConfigurationJson(), fromSettings)
	for name, value := range call.GetValues() {
		assertNoKey(t, "value "+name, value, fromSettings)
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
	assertNoKey(t, "the configuration json", cfg, fromSettings, fromKeychain)
	if _, found := host.GetPluginSettings(secretPluginID)["apiKey"]; found {
		t.Error("the settings the window reads hold a secret")
	}
}

// The environment a plugin process is launched with and its command line hold
// no key and no setting: neither the key in the store, nor one typed into the
// settings, nor one in the environment of the host.
func TestNoKeyIsInTheEnvironmentOrOnTheCommandLine(t *testing.T) {
	const fromHostEnv = "value-in-the-host-environment"
	for _, name := range []string{
		"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "NUMEN_AGENT_KEY", "NUMEN_EMBEDDING_KEY", "NUMEN_PROOFREADING_KEY",
	} {
		t.Setenv(name, fromHostEnv)
	}
	const fromSetting = "value-of-a-setting"
	store := newKeys(map[string]string{secretPluginID + "/apiKey": fromKeychain})
	host, _ := newSecretHost(t, store)
	host.SetSettings(map[string]map[string]any{secretPluginID: {declared: fromSettings, "model": fromSetting}})

	const launchName = "launch-name"
	vars, err := host.getLaunchVars(t.Context(), secretPluginID, nil, 4000, launchName)
	if err != nil {
		t.Fatal(err)
	}
	env := process.SanitizeEnvironment(os.Environ(), vars, []string{"onnxruntime"})
	args := process.GetPluginArgs(4000)
	for _, entry := range append(env, args...) {
		assertNoKey(t, "launch entry "+entry, entry, fromKeychain, fromSettings, fromHostEnv, fromSetting)
		if strings.HasPrefix(entry, "NUMEN_PLUGIN_CONFIG=") {
			t.Errorf("the settings are in the environment: %q", entry)
		}
	}
	if !slices.Contains(env, "NUMEN_PLUGIN_TOKEN="+launchName) {
		t.Fatalf("environment = %v, want the launch token in it", env)
	}
}

// A key store that fails to read a secret recorded as stored is an error naming
// it, and the plugin is still configured with everything else.
func TestAKeyStoreThatCannotBeReadFailsLoudly(t *testing.T) {
	store := newKeys(map[string]string{})
	store.failed = errLockedKeychain
	host, probe := newSecretHost(t, store)
	recordStored(host)

	err := host.NotifyPluginConfig(t.Context(), secretPluginID)
	if !errors.Is(err, errLockedKeychain) {
		t.Errorf("error = %v, want the error of the store", err)
	}
	if probe.last(t).GetValues()["model"] != "sonnet" {
		t.Error("the other settings were not passed on")
	}
	if _, dispatched := probe.last(t).GetValues()["apiKey"]; dispatched {
		t.Error("a key that could not be read was sent, which would clear the one the plugin holds")
	}
}

// A host with no key store configures a plugin that declares a secret, and
// fails to store a key for it.
func TestAHostWithoutAKeyStoreFailsToStoreASecret(t *testing.T) {
	host, probe := newSecretHost(t, nil)
	if err := host.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Errorf("configure: %v", err)
	}
	if probe.last(t).GetValues()["apiKey"] != "" {
		t.Error("a key came from nowhere")
	}
	if err := host.SetPluginSecret(t.Context(), secretPluginID, "apiKey", "value"); !errors.Is(err, pluginport.ErrNoKeyStore) {
		t.Errorf("set: error = %v, want port.ErrNoKeyStore", err)
	}
}

// A key set through the host is stored under the plugin and name, reaches the
// running plugin immediately, and is removed from it when deleted.
func TestAKeySetThroughTheHostReachesTheRunningPlugin(t *testing.T) {
	store := newKeys(map[string]string{})
	host, probe := newSecretHost(t, store)

	if err := host.SetPluginSecret(t.Context(), secretPluginID, "apiKey", fromKeychain); err != nil {
		t.Fatal(err)
	}
	if got, _ := store.hold(secretPluginID + "/apiKey"); got != fromKeychain {
		t.Errorf("stored %q", got)
	}
	if probe.last(t).GetValues()["apiKey"] != fromKeychain {
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

// Only a secret declared in the manifest can be stored, and only for a plugin
// known to the host.
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
	if len(store.item) != 0 {
		t.Errorf("the store holds %v", store.item)
	}
}

// A store that fails to save a key returns an error, and the plugin is not
// notified of a change.
func TestAKeyTheStoreCannotKeepIsAnError(t *testing.T) {
	store := newKeys(map[string]string{})
	store.failed = pluginport.ErrNoKeyStore
	host, probe := newSecretHost(t, store)

	if err := host.SetPluginSecret(t.Context(), secretPluginID, "apiKey", fromKeychain); !errors.Is(err, pluginport.ErrNoKeyStore) {
		t.Errorf("error = %v, want port.ErrNoKeyStore", err)
	}
	if len(probe.calls) != 0 {
		t.Error("the plugin was configured after a failed store")
	}
}

// A plugin that does not respond to Configure cannot receive its key, and the
// key change returns an error; a plugin without secrets that does not respond is
// not an error.
func TestAPluginThatCannotBeConfiguredHasNotBeenHandedItsKey(t *testing.T) {
	host, probe := newSecretHost(t, newKeys(map[string]string{}))
	probe.unimplemented = true
	if err := host.SetPluginSecret(t.Context(), secretPluginID, "apiKey", fromKeychain); err == nil {
		t.Error("a key the plugin never received was reported as delivered")
	}

	plainPlugin := host.plugins[secretPluginID]
	plainPlugin.Manifest.Secrets = nil
	host.plugins[secretPluginID] = plainPlugin
	if err := host.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Errorf("a plugin without secrets that does not answer Configure: %v", err)
	}
}

// The changes of one plugin's secrets are applied and delivered one at a time,
// so the plugin ends holding what the store holds.
func TestChangesOfASecretAreAppliedOneAtATime(t *testing.T) {
	store := newKeys(map[string]string{})
	store.entered = make(chan struct{}, 1)
	store.gate = make(chan struct{})
	host, probe := newSecretHost(t, store)

	setDone := make(chan error, 1)
	go func() { setDone <- host.SetPluginSecret(t.Context(), secretPluginID, "apiKey", fromKeychain) }()
	<-store.entered

	deleteDone := make(chan error, 1)
	go func() { deleteDone <- host.DeletePluginSecret(t.Context(), secretPluginID, "apiKey") }()
	select {
	case err := <-deleteDone:
		t.Fatalf("a delete ran while a set was in flight: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	close(store.gate)
	if err := <-setDone; err != nil {
		t.Fatal(err)
	}
	if err := <-deleteDone; err != nil {
		t.Fatal(err)
	}
	if _, found := store.hold(secretPluginID + "/apiKey"); found {
		t.Error("the store holds a key the delete removed")
	}
	if probe.last(t).GetValues()["apiKey"] != "" {
		t.Errorf("the plugin holds %v after the delete", probe.last(t).GetValues())
	}
}

// A Configure from elsewhere waits for the change of a secret in flight, so it
// cannot send the plugin the key the store held before the change.
func TestAConfigureWaitsForASecretChangeInFlight(t *testing.T) {
	store := newKeys(map[string]string{})
	store.entered = make(chan struct{}, 1)
	store.gate = make(chan struct{})
	host, probe := newSecretHost(t, store)

	setDone := make(chan error, 1)
	go func() { setDone <- host.SetPluginSecret(t.Context(), secretPluginID, "apiKey", fromKeychain) }()
	<-store.entered

	errCh := make(chan error, 1)
	go func() { errCh <- host.NotifyPluginConfig(t.Context(), secretPluginID) }()
	select {
	case err := <-errCh:
		t.Fatalf("a Configure ran during a secret change: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	close(store.gate)
	if err := <-setDone; err != nil {
		t.Fatal(err)
	}
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
	if probe.last(t).GetValues()["apiKey"] != fromKeychain {
		t.Errorf("the plugin ended holding %v", probe.last(t).GetValues())
	}
}

var errLockedKeychain = errors.New("the keychain is locked")

// recordStored records the apiKey secret as stored, the way the settings file
// holds the record.
func recordStored(host *Host) {
	host.SetSettings(map[string]map[string]any{secretPluginID: {pluginport.SettingStoredSecrets: []any{declared}}})
}

// untouchable is a key store that fails the test when it is reached.
type untouchable struct{ t *testing.T }

func (u untouchable) GetKey(context.Context, string) (string, error) {
	u.t.Error("the key store was read for a secret nobody stored")
	return "", pluginport.ErrNoKey
}

func (u untouchable) SetKey(context.Context, string, string) error {
	u.t.Error("the key store was written")
	return nil
}

func (u untouchable) DeleteKey(context.Context, string) error {
	u.t.Error("the key store was deleted from")
	return nil
}

// A secret that was never stored is delivered as empty without any access to
// the key store, at plugin start and on every setting change.
func TestASecretNobodyStoredPreservesTheKeyStoreAlone(t *testing.T) {
	host, probe := newSecretHost(t, untouchable{t})
	if err := host.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}
	if err := host.UpdatePluginSetting(t.Context(), secretPluginID, "model", "opus"); err != nil {
		t.Fatal(err)
	}
	if got, found := probe.last(t).GetValues()["apiKey"]; !found || got != "" {
		t.Errorf("apiKey = %q, %v for a key nobody stored", got, found)
	}
}

// A machine without a key store configures the plugin without the secret it
// recorded as stored, reports that once, and fails neither Configure nor a
// change of a setting.
func TestAMachineWithoutAKeyStoreDoesNotFailConfigure(t *testing.T) {
	store := newKeys(map[string]string{})
	store.failed = pluginport.ErrNoKeyStore
	host, probe := newSecretHost(t, store)
	recordStored(host)
	var reported []error
	host.SetErrorHandler(func(err error) { reported = append(reported, err) })

	if err := host.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Errorf("configure: %v", err)
	}
	if err := host.UpdatePluginSetting(t.Context(), secretPluginID, "model", "opus"); err != nil {
		t.Errorf("a change of a setting: %v", err)
	}
	call := probe.last(t)
	if _, dispatched := call.GetValues()["apiKey"]; dispatched {
		t.Error("a key that could not be read was sent, which would clear the one the plugin holds")
	}
	if call.GetValues()["model"] != "opus" {
		t.Errorf("values = %v, want the setting", call.GetValues())
	}
	if len(reported) != 1 || !errors.Is(reported[0], pluginport.ErrNoKeyStore) {
		t.Errorf("reported %v, want port.ErrNoKeyStore once", reported)
	}
}

// Storing a key records the name of its secret among the settings, never the
// key, and deleting it clears the record; neither the window nor the plugin
// sees the record.
func TestAStoredSecretIsRecordedByName(t *testing.T) {
	store := newKeys(map[string]string{})
	host, probe := newSecretHost(t, store)
	persisted := map[string]any{}
	// The settings file hands every write back to the host, as the container does.
	host.SetPersistHandler(func(pluginID, key string, value any) error {
		persisted[pluginID+"/"+key] = value
		return host.UpdatePluginSetting(t.Context(), pluginID, key, []any{"forged"})
	})
	record := secretPluginID + "/" + pluginport.SettingStoredSecrets

	set := make(chan error, 1)
	go func() { set <- host.SetPluginSecret(t.Context(), secretPluginID, declared, fromKeychain) }()
	select {
	case err := <-set:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("storing a key waits on the write of its record")
	}
	if got := host.getStoredSecretsLocked(secretPluginID); !slices.Equal(got, []string{declared}) {
		t.Errorf("the host holds the record %v, want the name of the secret", got)
	}
	if err := host.SetPluginSetting(t.Context(), secretPluginID, pluginport.SettingStoredSecrets, []any{declared}); !errors.Is(err, pluginport.ErrReservedSettingKey) {
		t.Errorf("a window writing the record: error = %v", err)
	}
	if got, ok := persisted[record].([]string); !ok || !slices.Equal(got, []string{declared}) {
		t.Errorf("recorded %#v, want the name of the secret", persisted[record])
	}
	if _, observed := host.GetPluginSettings(secretPluginID)[pluginport.SettingStoredSecrets]; observed {
		t.Error("the window sees the record")
	}
	if _, observedVal := probe.last(t).GetValues()[pluginport.SettingStoredSecrets]; observedVal {
		t.Error("the plugin sees the record")
	}

	if err := host.DeletePluginSecret(t.Context(), secretPluginID, declared); err != nil {
		t.Fatal(err)
	}
	if got, ok := persisted[record].([]string); !ok || len(got) != 0 {
		t.Errorf("recorded %#v after the delete, want none", persisted[record])
	}
	host.SetKeyStore(untouchable{t})
	if err := host.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}
}

// A record that cannot be persisted fails the store of the key.
func TestARecordThatCannotBePersistedFailsTheStore(t *testing.T) {
	host, _ := newSecretHost(t, newKeys(map[string]string{}))
	errFull := errors.New("the disk is full")
	host.SetPersistHandler(func(string, string, any) error { return errFull })
	if err := host.SetPluginSecret(t.Context(), secretPluginID, declared, fromKeychain); !errors.Is(err, errFull) {
		t.Errorf("error = %v, want the persist error", err)
	}
}
