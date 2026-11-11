package plugin

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/process"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
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
		return "", plugindomain.ErrNoKey
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
		return plugindomain.ErrNoKey
	}
	delete(k.item, name)
	return nil
}

// configured records the Configure calls a plugin receives.
type configured struct {
	wiretest.PluginLifecycle
	mu            sync.Mutex
	calls         []*v1.ConfigureRequest
	unimplemented bool
	shutdowns     int
	failure       error
	hold          chan struct{}
	entered       chan struct{}
	delay         time.Duration
	during        func()
}

func (c *configured) setFailure(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failure = err
}

func (c *configured) setHold(t *testing.T) (entered <-chan struct{}, release func()) {
	t.Helper()
	hold, arrivals := make(chan struct{}), make(chan struct{}, 8)
	c.mu.Lock()
	c.hold, c.entered = hold, arrivals
	c.mu.Unlock()
	var once sync.Once
	release = func() { once.Do(func() { close(hold) }) }
	t.Cleanup(release)
	return arrivals, release
}

func (c *configured) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.calls)
}

func (c *configured) Configure(
	ctx context.Context, req *connect.Request[v1.ConfigureRequest],
) (*connect.Response[v1.ConfigureResponse], error) {
	c.mu.Lock()
	hold, entered, delay, during := c.hold, c.entered, c.delay, c.during
	c.mu.Unlock()
	if during != nil {
		during()
	}
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
	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls = append(c.calls, req.Msg)
	if c.failure != nil {
		return nil, c.failure
	}
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
		if err := host.Teardown.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond); err != nil {
			t.Error(err)
		}
	})
	setPlugin(host, secretPluginID, manifest.Discovered{Manifest: manifest.Manifest{
		ID:      secretPluginID,
		Secrets: []manifest.SecretDeclaration{{Name: "apiKey", Title: "API key"}},
		Settings: []manifest.Setting{
			{Key: "model", Type: "string", Title: "Model", Default: "sonnet"},
		},
	}})
	sup := lifecycle.New(nil)
	sup.ActivateInProcess(testsupport.GetMachine().Now())
	putSupervisor(host, secretPluginID, sup)
	putEndpoint(host, secretPluginID, transport.Endpoint{
		Host: "127.0.0.1", Port: wiretest.GetPort(t, server), Token: "launch-name",
	})
	if store != nil {
		host.Config.SetKeyStore(store)
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

func allowOne(pluginID string) plugindomain.SecretSource {
	return plugindomain.SecretSource{List: func() ([]plugindomain.Secret, error) {
		return []plugindomain.Secret{{
			ID: secretID, Name: "Work", Group: "openrouter",
			Allowed: []plugindomain.DeclarationKey{{Plugin: pluginID, Name: declared}},
		}}, nil
	}}
}

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
	host.Config.SetSecretSource(allowOne(secretPluginID))

	if err := host.Config.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
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

func TestEmptyDeclarationSkipsKeyStoreRead(t *testing.T) {
	host, probe := newSecretHost(t, untouchable{t})
	if err := host.Config.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}
	if err := updateSetting(t.Context(), host, secretPluginID, "model", "opus"); err != nil {
		t.Fatal(err)
	}
	if got := listed(t, probe, declared); len(got) != 0 {
		t.Errorf("delivered %v, want no secret", got)
	}
}

// A secret the store has no value for is left out of the list.
func TestASecretWithNoValueInTheStoreIsLeftOut(t *testing.T) {
	host, probe := newSecretHost(t, newKeys(map[string]string{}))
	host.Config.SetSecretSource(allowOne(secretPluginID))
	if err := host.Config.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}
	if got := listed(t, probe, declared); len(got) != 0 {
		t.Errorf("delivered %v for a value nobody stored", got)
	}
}

// A key typed into the settings file under the name of a secret is not a key.
func TestAKeyTypedIntoTheSettingsIsNotPassedOn(t *testing.T) {
	host, probe := newSecretHost(t, newKeys(map[string]string{}))
	host.Settings.SetSettings(map[string]map[string]any{secretPluginID: {declared: fromSettings, "model": "opus"}})

	if err := host.Config.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
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
	cfg, err := host.Settings.GetPluginConfigJSON(secretPluginID)
	if err != nil {
		t.Fatal(err)
	}
	assertNoKey(t, "the configuration json", cfg, fromSettings, fromKeychain)
	if _, found := host.Settings.GetPluginSettings(secretPluginID)["apiKey"]; found {
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
	host.Config.SetSecretSource(allowOne(secretPluginID))
	host.Settings.SetSettings(map[string]map[string]any{secretPluginID: {declared: fromSettings, "model": fromSetting}})

	const launchName = "launch-name"
	vars, err := host.Processes.GetLaunchVars(t.Context(), secretPluginID, nil, host.Mode.GetVaultPath(), 4000, launchName)
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

func TestUnreadableKeyStoreFailsAndSendsEmptyDeclaration(t *testing.T) {
	store := newKeys(map[string]string{})
	store.failed = errLockedKeychain
	host, probe := newSecretHost(t, store)
	host.Config.SetSecretSource(allowOne(secretPluginID))

	err := host.Config.NotifyPluginConfig(t.Context(), secretPluginID)
	if !errors.Is(err, errLockedKeychain) {
		t.Errorf("error = %v, want the error of the store", err)
	}
	if probe.last(t).GetValues()["model"] != "sonnet" {
		t.Error("the other settings were not passed on")
	}
	if got := listed(t, probe, declared); len(got) != 0 {
		t.Errorf("the entry carries %v, want no value", got)
	}
}

// A machine without a key store configures the plugin with an empty list, reports that once, and
// fails neither Configure nor a change of a setting.
func TestAMachineWithoutAKeyStoreDoesNotFailConfigure(t *testing.T) {
	store := newKeys(map[string]string{})
	store.failed = plugindomain.ErrNoKeyStore
	host, probe := newSecretHost(t, store)
	host.Config.SetSecretSource(allowOne(secretPluginID))
	var reported []error
	host.Failures.SetErrorHandler(func(err error) { reported = append(reported, err) })

	if err := host.Config.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Errorf("configure: %v", err)
	}
	if err := updateSetting(t.Context(), host, secretPluginID, "model", "opus"); err != nil {
		t.Errorf("a change of a setting: %v", err)
	}
	if got := listed(t, probe, declared); len(got) != 0 {
		t.Errorf("delivered %v with no key store", got)
	}
	if probe.last(t).GetValues()["model"] != "opus" {
		t.Errorf("values = %v, want the setting", probe.last(t).GetValues())
	}
	if len(reported) != 1 || !errors.Is(reported[0], plugindomain.ErrNoKeyStore) {
		t.Errorf("reported %v, want port.ErrNoKeyStore once", reported)
	}
}

// untouchable is a key store that fails the test when it is reached.
type untouchable struct{ t *testing.T }

func (u untouchable) GetKey(context.Context, string) (string, error) {
	u.t.Error("the key store was read for a declaration nothing is allowed to")
	return "", plugindomain.ErrNoKey
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
	if err := host.Config.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}
	calls := len(probe.calls)

	if err := host.Config.Refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(probe.calls) != calls {
		t.Errorf("%d Configure calls after a refresh with nothing changed, want %d", len(probe.calls), calls)
	}

	host.Config.SetSecretSource(allowOne(secretPluginID))
	if err := host.Config.Refresh(t.Context()); err != nil {
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
	host.Config.SetSecretSource(allowOne(secretPluginID))
	if err := host.Config.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}

	err := applyChange(host, t.Context(), func() ([]string, error) {
		return []string{secretPluginID}, store.SetKey(t.Context(), "secret/"+secretID, "value-replaced")
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := listed(t, probe, declared); len(got) != 1 || got[0].GetValue() != "value-replaced" {
		t.Errorf("delivered %v, want the replaced value", got)
	}
}

// A refused operation configures no plugin.
func TestAnOperationThatFailsConfiguresNoPlugin(t *testing.T) {
	host, probe := newSecretHost(t, newKeys(map[string]string{}))
	if err := host.Config.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}
	calls := len(probe.calls)
	errRefused := errors.New("refused")

	err := applyChange(host, t.Context(), func() ([]string, error) { return []string{secretPluginID}, errRefused })
	if !errors.Is(err, errRefused) || len(probe.calls) != calls {
		t.Errorf("error = %v, %d calls, want the refusal and %d calls", err, len(probe.calls), calls)
	}
}

// A refresh and a Configure from elsewhere run one at a time.
func TestARefreshAndAConfigureRunOneAtATime(t *testing.T) {
	host, probe := newSecretHost(t, newKeys(map[string]string{"secret/" + secretID: fromKeychain}))
	source := newListing("Work")
	host.Config.SetSecretSource(source.source())
	entered, release := probe.setHold(t)

	refreshDone := make(chan error, 1)
	go func() { refreshDone <- host.Config.Refresh(t.Context()) }()
	<-entered
	configureDone := make(chan error, 1)
	go func() { configureDone <- host.Config.NotifyPluginConfig(t.Context(), secretPluginID) }()
	select {
	case err := <-configureDone:
		t.Fatalf("a Configure ran during a refresh: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	release()
	for _, result := range []chan error{refreshDone, configureDone} {
		if err := <-result; err != nil {
			t.Fatal(err)
		}
	}
	if probe.count() != 2 {
		t.Errorf("%d Configure calls, want 2", probe.count())
	}
}

// The declarations are listed for every installed plugin, with the group each declares.
func TestEveryDeclarationOfEveryInstalledPluginIsListed(t *testing.T) {
	host, _ := newSecretHost(t, nil)
	setPlugin(host, "md.numen.other", manifest.Discovered{Manifest: manifest.Manifest{
		ID: "md.numen.other",
		Secrets: []manifest.SecretDeclaration{
			{Name: "token", Title: "Token", Group: "mistral"},
		},
	}})
	want := []plugindomain.DeclarationKey{
		{Plugin: secretPluginID, Name: declared},
		{Plugin: "md.numen.other", Name: "token", Group: "mistral"},
	}
	if got := host.Config.ListDeclarations(); !slices.Equal(got, want) && !slices.Equal(got, []plugindomain.DeclarationKey{want[1], want[0]}) {
		t.Errorf("declarations = %v, want %v", got, want)
	}
}

// Permissions of a declaration that is gone are dropped after a scan, and not after a scan that had
// a problem, which may have hidden an installed plugin.
func TestPermissionsOfRemovedDeclarationsAreDroppedAfterAScan(t *testing.T) {
	host, _ := newSecretHost(t, nil)
	var pruned [][]plugindomain.DeclarationKey
	host.Config.SetSecretSource(plugindomain.SecretSource{Prune: func(declared []plugindomain.DeclarationKey) error {
		pruned = append(pruned, declared)
		return nil
	}})
	host.Config.hasScanned = true

	host.Config.pruneSecretPermissions()
	if len(pruned) != 1 || len(pruned[0]) != 1 || pruned[0][0].Plugin != secretPluginID {
		t.Fatalf("pruned with %v, want the declarations of the installed plugin", pruned)
	}

	host.Registry.set.mu.Lock()
	host.Registry.set.problems = append(host.Registry.set.problems, manifestjson.DiscoveryProblem{Dir: "/plugins/broken", Err: errors.New("unreadable")})
	host.Registry.set.mu.Unlock()
	host.Config.pruneSecretPermissions()
	if len(pruned) != 1 {
		t.Errorf("a scan with a problem pruned again: %v", pruned)
	}
}

// A plugin between its activation and the Configure of that activation is not due one: the
// Configure of the activation reads the list as it is, so a refresh sends nothing to it.
func TestAPluginAwaitingItsActivationConfigureIsNotDueOne(t *testing.T) {
	host, probe := newSecretHost(t, newKeys(map[string]string{"secret/" + secretID: fromKeychain}))
	host.Config.SetSecretSource(allowOne(secretPluginID))
	host.Config.delivery.AwaitFirstConfigure(secretPluginID)

	if err := host.Config.Refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	if probe.count() != 0 {
		t.Errorf("the refresh sent %d Configure calls to a plugin that awaits its activation Configure", probe.count())
	}
}
