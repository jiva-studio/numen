package plugin

import (
	"context"
	"errors"
	"io"
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
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugintest"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/process"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifestjson"

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

func newSecretHost(t *testing.T, store pluginport.KeyStore) (*Host, *configured) {
	t.Helper()
	probe := &configured{}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(probe))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	host := NewHost(testsupport.GetMachine(), "desktop.editor")
	t.Cleanup(func() {
		if err := host.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond); err != nil {
			t.Error(err)
		}
	})
	host.plugins[secretPluginID] = manifest.Discovered{Manifest: pluginport.PluginManifest{
		ID:      secretPluginID,
		Secrets: []pluginport.PluginSecret{{Name: "apiKey", Title: "API key"}},
		Settings: []manifest.Setting{
			{Key: "model", Type: "string", Title: "Model", Default: "sonnet"},
		},
	}}
	sup := lifecycle.New(testsupport.GetMachine())
	sup.ActivateInProcess(testsupport.GetMachine().Now())
	host.supervisors[secretPluginID] = sup
	host.endpoints[secretPluginID] = transport.Endpoint{
		Host: "127.0.0.1", Port: plugintest.GetPort(t, server), Token: "launch-name",
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

// secretID is the id of the one secret the tests of this file list.
const secretID = "s0a1b2c3d"

// allowOne is a source that lists one secret, allowed to the declaration apiKey of the plugin.
func allowOne(pluginID string) pluginport.SecretSource {
	return pluginport.SecretSource{List: func() ([]pluginport.Secret, error) {
		return []pluginport.Secret{{
			ID: secretID, Name: "Work", Group: "openrouter",
			Allowed: []pluginport.SecretPermission{{Plugin: pluginID, Name: declared}},
		}}, nil
	}}
}

// listed returns the secrets the last Configure call delivers to the declaration.
func listed(t *testing.T, probe *configured, name string) []*v1.SecretValue {
	t.Helper()
	for _, entry := range probe.last(t).GetSecrets() {
		if entry.GetName() == name {
			return entry.GetValues()
		}
	}
	t.Fatalf("the call carries no entry for %s: %v", name, probe.last(t).GetSecrets())
	return nil
}

// The value arrives in the entry of its declaration and nowhere else in the call.
func TestTheValueReachesThePluginOnlyInTheEntryOfItsDeclaration(t *testing.T) {
	host, probe := newSecretHost(t, newKeys(map[string]string{"secret/" + secretID: fromKeychain}))
	host.SetSecretSource(allowOne(secretPluginID))

	if err := host.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}
	call := probe.last(t)
	got := listed(t, probe, declared)
	if len(got) != 1 || got[0].GetValue() != fromKeychain || got[0].GetId() != secretID || got[0].GetName() != "Work" || got[0].GetGroup() != "openrouter" {
		t.Errorf("delivered %v, want the allowed secret with its value", got)
	}
	assertNoKey(t, "the configuration json", call.GetConfigurationJson(), fromKeychain)
	for name, value := range call.GetValues() {
		assertNoKey(t, "value "+name, value, fromKeychain)
	}
	if call.GetValues()["model"] != "sonnet" {
		t.Errorf("values = %v, want the model setting", call.GetValues())
	}
}

// A declaration nothing is allowed to is delivered as an entry with no values, without any read of
// the key store.
func TestADeclarationWithNothingAllowedIsDeliveredEmptyAndTheStoreIsNotRead(t *testing.T) {
	host, probe := newSecretHost(t, untouchable{t})
	if err := host.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}
	if err := host.UpdatePluginSetting(t.Context(), secretPluginID, "model", "opus"); err != nil {
		t.Fatal(err)
	}
	if got := listed(t, probe, declared); len(got) != 0 {
		t.Errorf("delivered %v, want no secret", got)
	}
}

// A secret the store has no value for is left out of the list.
func TestASecretWithNoValueInTheStoreIsLeftOut(t *testing.T) {
	host, probe := newSecretHost(t, newKeys(map[string]string{}))
	host.SetSecretSource(allowOne(secretPluginID))
	if err := host.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}
	if got := listed(t, probe, declared); len(got) != 0 {
		t.Errorf("delivered %v for a value nobody stored", got)
	}
}

// A key typed into the settings file under the name of a secret is not a key.
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

// The environment a plugin process is launched with and its command line hold no key and no
// setting: neither the key in the store, nor one typed into the settings, nor one in the
// environment of the host.
func TestNoKeyIsInTheEnvironmentOrOnTheCommandLine(t *testing.T) {
	const fromHostEnv = "value-in-the-host-environment"
	for _, name := range []string{
		"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "NUMEN_AGENT_KEY", "NUMEN_EMBEDDING_KEY", "NUMEN_PROOFREADING_KEY",
	} {
		t.Setenv(name, fromHostEnv)
	}
	const fromSetting = "value-of-a-setting"
	host, _ := newSecretHost(t, newKeys(map[string]string{"secret/" + secretID: fromKeychain}))
	host.SetSecretSource(allowOne(secretPluginID))
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

var errLockedKeychain = errors.New("the keychain is locked")

// A key store that fails to read an allowed secret is an error naming it, the plugin is still
// configured with everything else, and the declaration is not carried, so the plugin keeps the
// secrets it has.
func TestAKeyStoreThatCannotBeReadFailsLoudlyAndTheDeclarationIsNotCarried(t *testing.T) {
	store := newKeys(map[string]string{})
	store.failed = errLockedKeychain
	host, probe := newSecretHost(t, store)
	host.SetSecretSource(allowOne(secretPluginID))

	err := host.NotifyPluginConfig(t.Context(), secretPluginID)
	if !errors.Is(err, errLockedKeychain) {
		t.Errorf("error = %v, want the error of the store", err)
	}
	if probe.last(t).GetValues()["model"] != "sonnet" {
		t.Error("the other settings were not passed on")
	}
	if got := probe.last(t).GetSecrets(); len(got) != 0 {
		t.Errorf("the call carries %v, want no entry for a declaration that could not be read", got)
	}
}

// A machine without a key store configures the plugin with an empty list, reports that once, and
// fails neither Configure nor a change of a setting.
func TestAMachineWithoutAKeyStoreDoesNotFailConfigure(t *testing.T) {
	store := newKeys(map[string]string{})
	store.failed = pluginport.ErrNoKeyStore
	host, probe := newSecretHost(t, store)
	host.SetSecretSource(allowOne(secretPluginID))
	var reported []error
	host.SetErrorHandler(func(err error) { reported = append(reported, err) })

	if err := host.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Errorf("configure: %v", err)
	}
	if err := host.UpdatePluginSetting(t.Context(), secretPluginID, "model", "opus"); err != nil {
		t.Errorf("a change of a setting: %v", err)
	}
	if got := listed(t, probe, declared); len(got) != 0 {
		t.Errorf("delivered %v with no key store", got)
	}
	if probe.last(t).GetValues()["model"] != "opus" {
		t.Errorf("values = %v, want the setting", probe.last(t).GetValues())
	}
	if len(reported) != 1 || !errors.Is(reported[0], pluginport.ErrNoKeyStore) {
		t.Errorf("reported %v, want port.ErrNoKeyStore once", reported)
	}
}

// untouchable is a key store that fails the test when it is reached.
type untouchable struct{ t *testing.T }

func (u untouchable) GetKey(context.Context, string) (string, error) {
	u.t.Error("the key store was read for a declaration nothing is allowed to")
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

// A refresh configures a plugin only when its list of secrets differs from the one it was given.
func TestARefreshConfiguresOnlyAPluginWhoseListChanged(t *testing.T) {
	store := newKeys(map[string]string{"secret/" + secretID: fromKeychain})
	host, probe := newSecretHost(t, store)
	if err := host.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}
	calls := len(probe.calls)

	if err := host.RefreshSecrets(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(probe.calls) != calls {
		t.Errorf("%d Configure calls after a refresh with nothing changed, want %d", len(probe.calls), calls)
	}

	host.SetSecretSource(allowOne(secretPluginID))
	if err := host.RefreshSecrets(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(probe.calls) != calls+1 || len(listed(t, probe, declared)) != 1 {
		t.Errorf("%d Configure calls after an allow, want %d with the secret", len(probe.calls), calls+1)
	}
}

// A plugin named by an operation is configured although its list did not change, so it takes a new value.
func TestAPluginNamedByAnOperationIsConfiguredAgain(t *testing.T) {
	store := newKeys(map[string]string{"secret/" + secretID: fromKeychain})
	host, probe := newSecretHost(t, store)
	host.SetSecretSource(allowOne(secretPluginID))
	if err := host.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}

	err := host.ApplySecretChange(t.Context(), func() ([]string, error) {
		return []string{secretPluginID}, store.SetKey(t.Context(), "secret/"+secretID, "value-replaced")
	})
	if err != nil {
		t.Fatal(err)
	}
	plugintest.WaitUntil(t, "the replaced value is delivered", func() bool {
		got := listed(t, probe, declared)
		return len(got) == 1 && got[0].GetValue() == "value-replaced"
	})
}

// A refused operation configures no plugin.
func TestAnOperationThatFailsConfiguresNoPlugin(t *testing.T) {
	host, probe := newSecretHost(t, newKeys(map[string]string{}))
	if err := host.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}
	calls := len(probe.calls)
	errRefused := errors.New("refused")

	err := host.ApplySecretChange(t.Context(), func() ([]string, error) { return []string{secretPluginID}, errRefused })
	if !errors.Is(err, errRefused) || len(probe.calls) != calls {
		t.Errorf("error = %v, %d calls, want the refusal and %d calls", err, len(probe.calls), calls)
	}
}

// Two operations run one at a time, and a Configure from elsewhere waits for the one in flight.
func TestOperationsAndConfigureRunOneAtATime(t *testing.T) {
	host, _ := newSecretHost(t, newKeys(map[string]string{}))
	entered, gate := make(chan struct{}, 1), make(chan struct{})

	firstDone := make(chan error, 1)
	go func() {
		firstDone <- host.ApplySecretChange(t.Context(), func() ([]string, error) {
			entered <- struct{}{}
			<-gate
			return nil, nil
		})
	}()
	<-entered

	secondDone := make(chan error, 1)
	go func() {
		secondDone <- host.ApplySecretChange(t.Context(), func() ([]string, error) { return nil, nil })
	}()
	configureDone := make(chan error, 1)
	go func() { configureDone <- host.NotifyPluginConfig(t.Context(), secretPluginID) }()
	select {
	case err := <-secondDone:
		t.Fatalf("an operation ran while another was in flight: %v", err)
	case err := <-configureDone:
		t.Fatalf("a Configure ran during an operation: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	close(gate)
	for _, result := range []chan error{firstDone, secondDone, configureDone} {
		if err := <-result; err != nil {
			t.Fatal(err)
		}
	}
}

// The declarations are listed for every installed plugin, with the group each declares.
func TestEveryDeclarationOfEveryInstalledPluginIsListed(t *testing.T) {
	host, _ := newSecretHost(t, nil)
	host.plugins["md.numen.other"] = manifest.Discovered{Manifest: pluginport.PluginManifest{
		ID: "md.numen.other",
		Secrets: []pluginport.PluginSecret{
			{Name: "token", Title: "Token", Group: "mistral"},
		},
	}}
	want := []pluginport.SecretPermission{
		{Plugin: secretPluginID, Name: declared},
		{Plugin: "md.numen.other", Name: "token", Group: "mistral"},
	}
	if got := host.ListDeclarations(); !slices.Equal(got, want) && !slices.Equal(got, []pluginport.SecretPermission{want[1], want[0]}) {
		t.Errorf("declarations = %v, want %v", got, want)
	}
}

// Permissions of a declaration that is gone are dropped after a scan, and not after a scan that had
// a problem, which may have hidden an installed plugin.
func TestPermissionsOfRemovedDeclarationsAreDroppedAfterAScan(t *testing.T) {
	host, _ := newSecretHost(t, nil)
	var pruned [][]pluginport.SecretPermission
	host.SetSecretSource(pluginport.SecretSource{Prune: func(declared []pluginport.SecretPermission) error {
		pruned = append(pruned, declared)
		return nil
	}})
	host.hasScanned = true

	host.pruneSecretPermissions()
	if len(pruned) != 1 || len(pruned[0]) != 1 || pruned[0][0].Plugin != secretPluginID {
		t.Fatalf("pruned with %v, want the declarations of the installed plugin", pruned)
	}

	host.problems = append(host.problems, manifestjson.DiscoveryProblem{Dir: "/plugins/broken", Err: errors.New("unreadable")})
	host.pruneSecretPermissions()
	if len(pruned) != 1 {
		t.Errorf("a scan with a problem pruned again: %v", pruned)
	}
}

// The signature of a list names the declarations and the secrets and holds no value.
func TestTheSignatureOfAListContainsNoValue(t *testing.T) {
	entries := []*v1.ConfigureSecrets{{
		Name: "apiKey", Group: "openrouter",
		Values: []*v1.SecretValue{{Id: secretID, Name: "Work", Group: "openrouter", Value: fromKeychain}},
	}}
	signature := getSecretsSignature(entries)
	assertNoKey(t, "the signature", signature, fromKeychain)
	entries[0].Values[0].Value = "another"
	if getSecretsSignature(entries) != signature {
		t.Error("the signature follows the value")
	}
	entries[0].Values[0].Name = "Home"
	if getSecretsSignature(entries) == signature {
		t.Error("the signature ignores the name of a secret")
	}
}

// A plugin that restarts, and whose first Configure fails, is configured again by the next refresh
// although its list of secrets did not change.
func TestARestartedPluginWhoseFirstConfigureFailedIsConfiguredByTheNextRefresh(t *testing.T) {
	host, probe := newSecretHost(t, newKeys(map[string]string{"secret/" + secretID: fromKeychain}))
	host.SetSecretSource(allowOne(secretPluginID))
	if err := host.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}

	probe.mu.Lock()
	probe.unimplemented = true
	probe.mu.Unlock()
	port := host.endpoints[secretPluginID].Port
	sleeper := plugintest.StartSleepProcess(t).Process
	host.SetLauncher(func(context.Context, manifest.Discovered, int, string) (*os.Process, io.ReadCloser, error) {
		return sleeper, plugintest.GetReadyOutput(port), nil
	})
	p := host.plugins[secretPluginID]
	p.Manifest.Backend = pluginport.PluginBackend{Entrypoint: "bin/plugin"}
	restarted := lifecycle.New(testsupport.GetMachine())
	restarted.SetActivating()
	if _, err := host.runActivation(t.Context(), p, restarted); err != nil {
		t.Fatal(err)
	}

	probe.mu.Lock()
	probe.unimplemented = false
	before := len(probe.calls)
	probe.mu.Unlock()
	if err := host.RefreshSecrets(t.Context()); err != nil {
		t.Fatal(err)
	}
	probe.mu.Lock()
	defer probe.mu.Unlock()
	if len(probe.calls) != before+1 {
		t.Errorf("the refresh sent %d Configure calls to the restarted plugin, want 1", len(probe.calls)-before)
	}
}
