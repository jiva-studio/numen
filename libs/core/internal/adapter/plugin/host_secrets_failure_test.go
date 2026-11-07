package plugin

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugintest"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	pluginport "github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

// collectFailures records the failures the host passes on.
type collectFailures struct {
	mu       sync.Mutex
	failures []pluginport.PluginFailure
}

func (c *collectFailures) add(failure pluginport.PluginFailure) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures = append(c.failures, failure)
}

func (c *collectFailures) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.failures)
}

// collectErrors records the errors the host reports outside a call.
type collectErrors struct {
	mu    sync.Mutex
	texts []string
}

func (c *collectErrors) add(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.texts = append(c.texts, err.Error())
}

func (c *collectErrors) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.texts)
}

// listing is a secret source with one secret, allowed to the plugins it names, whose name and group
// change between reads.
type listing struct {
	mu      sync.Mutex
	name    string
	group   string
	plugins []string
}

func newListing(name string) *listing {
	return &listing{name: name, group: "openrouter", plugins: []string{secretPluginID}}
}

func (l *listing) rename(name string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.name = name
}

func (l *listing) current() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.name
}

func (l *listing) regroup(group string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.group = group
}

// allow lets the plugin read the secret too.
func (l *listing) allow(pluginID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.plugins = append(l.plugins, pluginID)
}

func (l *listing) source() pluginport.SecretSource {
	return pluginport.SecretSource{List: func() ([]pluginport.Secret, error) {
		l.mu.Lock()
		defer l.mu.Unlock()
		var allowed []pluginport.SecretPermission
		for _, id := range l.plugins {
			allowed = append(allowed, pluginport.SecretPermission{Plugin: id, Name: declared})
		}
		return []pluginport.Secret{{ID: secretID, Name: l.name, Group: l.group, Allowed: allowed}}, nil
	}}
}

// renameBy returns an operation that renames the secret and names no reader.
func (l *listing) renameBy(name string) func() ([]string, error) {
	return func() ([]string, error) {
		l.rename(name)
		return nil, nil
	}
}

// deliveredName returns the name of the secret the last Configure call delivers to the declaration.
func deliveredName(t *testing.T, probe *configured) string {
	t.Helper()
	got := listed(t, probe, declared)
	if len(got) != 1 {
		t.Fatalf("delivered %v, want one secret", got)
	}
	return got[0].GetName()
}

// newConfiguredHost returns a host whose plugin was given the secret named "Work".
func newConfiguredHost(t *testing.T) (*Host, *configured, *listing) {
	t.Helper()
	host, probe := newSecretHost(t, newKeys(map[string]string{"secret/" + secretID: fromKeychain}))
	source := newListing("Work")
	host.SetSecretSource(source.source())
	if err := host.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}
	return host, probe, source
}

// prepareActivation makes the plugin ready for an activation and returns a function that runs it.
// With dormant set the plugin is first not running, as before its first start.
func prepareActivation(t *testing.T, host *Host, dormant bool) func() error {
	t.Helper()
	activate := prepareActivationWith(t, host, dormant)
	return func() error { return activate(t.Context()) }
}

// prepareActivationWith is prepareActivation for an activation that runs with the given context.
func prepareActivationWith(t *testing.T, host *Host, dormant bool) func(ctx context.Context) error {
	t.Helper()
	port := host.endpoints[secretPluginID].Port
	sleeper := plugintest.StartSleepProcess(t).Process
	host.SetLauncher(func(context.Context, manifest.Discovered, int, string) (*os.Process, io.ReadCloser, error) {
		return sleeper, plugintest.GetReadyOutput(port), nil
	})
	p := host.plugins[secretPluginID]
	p.Manifest.Backend = pluginport.PluginBackend{Entrypoint: "bin/plugin"}
	activating := lifecycle.New(testsupport.GetMachine())
	activating.SetActivating()
	if dormant {
		delete(host.endpoints, secretPluginID)
		host.supervisors[secretPluginID] = activating
	}
	return func(ctx context.Context) error {
		_, err := host.runActivation(ctx, p, activating)
		return err
	}
}

// addSecretPlugin adds a second running plugin that declares apiKey, lets it read the secret of the
// listing, and returns the probe that records its Configure calls.
func addSecretPlugin(t *testing.T, host *Host, source *listing, pluginID string) *configured {
	t.Helper()
	probe := &configured{}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(probe))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	host.plugins[pluginID] = manifest.Discovered{Manifest: pluginport.PluginManifest{
		ID:      pluginID,
		Secrets: []pluginport.PluginSecret{{Name: declared, Title: "API key"}},
	}}
	sup := lifecycle.New(testsupport.GetMachine())
	sup.ActivateInProcess(testsupport.GetMachine().Now())
	host.supervisors[pluginID] = sup
	host.endpoints[pluginID] = transport.Endpoint{Host: "127.0.0.1", Port: plugintest.GetPort(t, server), Token: "second"}
	source.allow(pluginID)
	return probe
}

// A plugin whose Configure fails does not undo the operation. The operation returns an error that
// names the plugin and holds no value, even when the plugin put the value in its own error text. The
// plugin is not marked: no failure is passed on and it may be started again.
func TestAFailedConfigureKeepsTheChangeAndNamesThePluginWithoutTheValue(t *testing.T) {
	host, probe, source := newConfiguredHost(t)
	failed := &collectFailures{}
	host.SetOnPluginFailed(failed.add)
	probe.setFailure(connect.NewError(connect.CodeInternal, errors.New("rejected the key "+fromKeychain)))

	committed := false
	err := host.ApplySecretChange(t.Context(), func() ([]string, error) {
		committed = true
		source.rename("Renamed")
		return nil, nil
	})

	if !committed {
		t.Error("the operation did not run")
	}
	if err == nil {
		t.Fatal("the operation returned nil although the plugin did not accept the list")
	}
	if !strings.Contains(err.Error(), secretPluginID) {
		t.Errorf("the error %q does not name the plugin", err)
	}
	assertNoKey(t, "the returned error", err.Error(), fromKeychain)
	if failed.count() != 0 {
		t.Errorf("%d failures passed on, want none", failed.count())
	}
	sup := host.supervisors[secretPluginID]
	if sup.IsActivationFailed() {
		t.Error("the plugin is marked as failed")
	}
	if err := sup.CheckActivationAllowed(testsupport.GetMachine().Now()); err != nil {
		t.Errorf("the plugin may not be started again: %v", err)
	}
	for _, info := range host.GetPluginInfos() {
		if info.ID == secretPluginID && info.Failure != nil {
			t.Errorf("the plugin page shows %+v", info.Failure)
		}
	}
}

// A refresh does not send again the list that a send just failed with, so a failure reaches the
// person once, with the operation that caused it.
func TestARefreshAfterAFailedSendDoesNotSendTheSameListAgain(t *testing.T) {
	host, probe, source := newConfiguredHost(t)
	reported := &collectErrors{}
	host.SetErrorHandler(reported.add)
	probe.setFailure(connect.NewError(connect.CodeInternal, errors.New("the plugin cannot take it")))
	if err := host.ApplySecretChange(t.Context(), source.renameBy("Renamed")); err == nil {
		t.Fatal("the operation returned nil although the plugin did not accept the list")
	}
	probe.setFailure(nil)
	calls := probe.count()

	if err := host.RefreshSecrets(t.Context()); err != nil {
		t.Errorf("the refresh returned %v, want the failure to reach the person once", err)
	}
	if probe.count() != calls {
		t.Errorf("%d Configure calls after the refresh, want %d: the list that failed was sent again", probe.count(), calls)
	}
	if got := reported.count(); got != 0 {
		t.Errorf("%d failures reported besides the one the operation returned", got)
	}
}

// A refresh sends to a plugin whose send failed once its list differs from the one that failed.
func TestARefreshSendsToAFailedPluginWhoseListChanged(t *testing.T) {
	host, probe, source := newConfiguredHost(t)
	probe.setFailure(connect.NewError(connect.CodeInternal, errors.New("the plugin cannot take it")))
	if err := host.ApplySecretChange(t.Context(), source.renameBy("Renamed")); err == nil {
		t.Fatal("the operation returned nil although the plugin did not accept the list")
	}
	probe.setFailure(nil)
	calls := probe.count()
	source.rename("Renamed again")

	if err := host.RefreshSecrets(t.Context()); err != nil {
		t.Fatal(err)
	}
	if probe.count() != calls+1 || deliveredName(t, probe) != "Renamed again" {
		t.Errorf("%d Configure calls, last list %q, want %d and the list that changed",
			probe.count(), deliveredName(t, probe), calls+1)
	}
}

// An operation that names the plugin sends to it although its last send failed with the same list,
// so the plugin takes a new value.
func TestAnOperationThatNamesAFailedPluginSendsToIt(t *testing.T) {
	host, probe, source := newConfiguredHost(t)
	probe.setFailure(connect.NewError(connect.CodeInternal, errors.New("the plugin cannot take it")))
	if err := host.ApplySecretChange(t.Context(), source.renameBy("Renamed")); err == nil {
		t.Fatal("the operation returned nil although the plugin did not accept the list")
	}
	probe.setFailure(nil)
	calls := probe.count()

	err := host.ApplySecretChange(t.Context(), func() ([]string, error) { return []string{secretPluginID}, nil })
	if err != nil {
		t.Fatal(err)
	}
	if probe.count() != calls+1 {
		t.Errorf("%d Configure calls, want %d", probe.count(), calls+1)
	}
}

// A plugin that restarts is sent the list that failed before, although the list did not change, and
// a refresh then sends nothing more.
func TestARestartedPluginIsSentTheListThatFailedBefore(t *testing.T) {
	host, probe, source := newConfiguredHost(t)
	probe.setFailure(connect.NewError(connect.CodeInternal, errors.New("the plugin cannot take it")))
	if err := host.ApplySecretChange(t.Context(), source.renameBy("Renamed")); err == nil {
		t.Fatal("the operation returned nil although the plugin did not accept the list")
	}
	probe.setFailure(nil)
	calls := probe.count()

	if err := prepareActivation(t, host, false)(); err != nil {
		t.Fatal(err)
	}
	if probe.count() != calls+1 || deliveredName(t, probe) != "Renamed" {
		t.Fatalf("%d Configure calls after the restart, last list %q, want %d and the list that failed",
			probe.count(), deliveredName(t, probe), calls+1)
	}
	if err := host.RefreshSecrets(t.Context()); err != nil {
		t.Fatal(err)
	}
	if probe.count() != calls+1 {
		t.Errorf("%d Configure calls after a refresh, want %d", probe.count(), calls+1)
	}
}

// The activation of a plugin sends to it although an earlier send failed with the same list, and
// when that one fails too, the person is notified once and a bare refresh does not send again.
func TestAFailedActivationConfigureIsReportedOnceAndNotSentAgainByARefresh(t *testing.T) {
	host, probe, source := newConfiguredHost(t)
	reported := &collectErrors{}
	host.SetErrorHandler(reported.add)
	probe.setFailure(connect.NewError(connect.CodeInternal, errors.New("the plugin cannot take it")))
	calls := probe.count()

	if err := prepareActivation(t, host, false)(); err != nil {
		t.Fatal(err)
	}
	if probe.count() != calls+1 || reported.count() != 1 {
		t.Fatalf("%d Configure calls and %d failures reported by the activation, want %d and 1",
			probe.count(), reported.count(), calls+1)
	}
	probe.setFailure(nil)
	if err := host.RefreshSecrets(t.Context()); err != nil {
		t.Errorf("the refresh returned %v", err)
	}
	if probe.count() != calls+1 || reported.count() != 1 {
		t.Errorf("%d Configure calls and %d failures reported after a bare refresh, want %d and 1",
			probe.count(), reported.count(), calls+1)
	}

	source.rename("Renamed")
	if err := host.RefreshSecrets(t.Context()); err != nil {
		t.Fatal(err)
	}
	if probe.count() != calls+2 || deliveredName(t, probe) != "Renamed" {
		t.Errorf("%d Configure calls, last list %q, want %d and the list that changed",
			probe.count(), deliveredName(t, probe), calls+2)
	}
}

// The next operation that changes the list of a plugin whose Configure failed sends again, and the
// plugin ends with the list of that operation.
func TestTheNextOperationThatChangesTheListOfAFailedPluginSendsAgain(t *testing.T) {
	host, probe, source := newConfiguredHost(t)
	probe.setFailure(connect.NewError(connect.CodeInternal, errors.New("the plugin cannot take it")))
	if err := host.ApplySecretChange(t.Context(), source.renameBy("Renamed")); err == nil {
		t.Fatal("the operation returned nil although the plugin did not accept the list")
	}
	probe.setFailure(nil)
	calls := probe.count()

	if err := host.ApplySecretChange(t.Context(), source.renameBy("Renamed again")); err != nil {
		t.Fatal(err)
	}
	if probe.count() != calls+1 || deliveredName(t, probe) != "Renamed again" {
		t.Errorf("%d Configure calls, last list %q, want %d and the list of the second operation",
			probe.count(), deliveredName(t, probe), calls+1)
	}
}

// An operation that leaves the list of a failing plugin as it is does not send to it, so it neither
// fails nor waits for it.
func TestAnOperationThatDoesNotChangeTheListOfAFailingPluginDoesNotSendToIt(t *testing.T) {
	host, probe, source := newConfiguredHost(t)
	probe.setFailure(connect.NewError(connect.CodeInternal, errors.New("the plugin cannot take it")))
	if err := host.ApplySecretChange(t.Context(), source.renameBy("Renamed")); err == nil {
		t.Fatal("the operation returned nil although the plugin did not accept the list")
	}
	calls := probe.count()

	if err := host.ApplySecretChange(t.Context(), func() ([]string, error) { return nil, nil }); err != nil {
		t.Errorf("an operation that does not concern the plugin returned %v", err)
	}
	if probe.count() != calls {
		t.Errorf("%d Configure calls, want %d: the plugin was sent to by an operation that does not concern it", probe.count(), calls)
	}
}

// A plugin that never answers delays the operation by the Configure timeout, once, and the
// operation returns an error that names it and says the call timed out. The operations that follow,
// among them the refresh the operation's own settings event causes, send nothing to the plugin and
// are not delayed.
func TestAHungConfigureDelaysTheOperationByTheTimeoutOnce(t *testing.T) {
	host, probe, source := newConfiguredHost(t)
	entered, _ := probe.setHold(t)

	result := make(chan error, 1)
	ctx := t.Context()
	go func() { result <- host.ApplySecretChange(ctx, source.renameBy("Renamed")) }()
	select {
	case <-result:
		t.Fatal("the operation returned before the Configure timeout")
	case <-time.After(configureCallTimeout / 2):
	}
	var err error
	select {
	case err = <-result:
	case <-time.After(configureCallTimeout + 2*time.Second):
		t.Fatal("the operation waits longer than the Configure timeout")
	}
	if len(entered) != 1 {
		t.Errorf("the plugin received %d calls, want 1", len(entered))
	}
	if err == nil || !strings.Contains(err.Error(), secretPluginID) || !strings.Contains(err.Error(), "timed out") {
		t.Errorf("the error = %v, want one that names the plugin and says the call timed out", err)
	}

	followUps := map[string]func() error{
		"the refresh after the operation": func() error { return host.RefreshSecrets(ctx) },
		"an operation that concerns no plugin": func() error {
			return host.ApplySecretChange(ctx, func() ([]string, error) { return nil, nil })
		},
	}
	for name, run := range followUps {
		followUp := make(chan error, 1)
		go func() { followUp <- run() }()
		select {
		case err := <-followUp:
			if err != nil {
				t.Errorf("%s returned %v", name, err)
			}
		case <-time.After(configureCallTimeout / 2):
			t.Fatalf("%s waits for the plugin that did not answer", name)
		}
	}
	if len(entered) != 1 {
		t.Errorf("the plugin received %d calls in all, want 1", len(entered))
	}
}

// A Configure of a plugin that never answers takes the timeout once, not twice.
func TestNotifyingAHungPluginWaitsForTheTimeoutOnce(t *testing.T) {
	host, probe, _ := newConfiguredHost(t)
	probe.setHold(t)

	result := make(chan error, 1)
	ctx := t.Context()
	go func() { result <- host.NotifyPluginConfig(ctx, secretPluginID) }()
	var err error
	select {
	case err = <-result:
	case <-time.After(configureCallTimeout + 2*time.Second):
		t.Fatal("the call waits longer than the Configure timeout")
	}
	if err == nil || !strings.Contains(err.Error(), secretPluginID) || !strings.Contains(err.Error(), "timed out") {
		t.Errorf("the error = %v, want one that names the plugin and says the call timed out", err)
	}
}

// The operation returns after the plugin answered, so the plugin holds the new list by then.
func TestTheOperationReturnsOnlyAfterThePluginAnswered(t *testing.T) {
	host, probe, source := newConfiguredHost(t)
	probe.mu.Lock()
	probe.delay = 300 * time.Millisecond
	probe.mu.Unlock()
	calls := probe.count()

	if err := host.ApplySecretChange(t.Context(), source.renameBy("Renamed")); err != nil {
		t.Fatal(err)
	}
	if probe.count() != calls+1 || deliveredName(t, probe) != "Renamed" {
		t.Errorf("%d Configure calls when the operation returned, want %d with the new list", probe.count(), calls+1)
	}
}

// The lock of the secret operations is held while the Configure call is on its way, so a second
// operation changes the list only after the first one's call was answered.
func TestTheLockIsHeldWhileTheConfigureIsSent(t *testing.T) {
	host, probe, source := newConfiguredHost(t)
	entered, release := probe.setHold(t)
	calls := probe.count()

	first := make(chan error, 1)
	go func() { first <- host.ApplySecretChange(t.Context(), source.renameBy("First")) }()
	<-entered

	var secondRan atomic.Bool
	second := make(chan error, 1)
	go func() {
		second <- host.ApplySecretChange(t.Context(), func() ([]string, error) {
			secondRan.Store(true)
			source.rename("Second")
			return nil, nil
		})
	}()
	time.Sleep(150 * time.Millisecond)
	if secondRan.Load() {
		t.Error("a second operation ran while the Configure call of the first was on its way")
	}

	release()
	for _, result := range []chan error{first, second} {
		if err := <-result; err != nil {
			t.Fatal(err)
		}
	}
	if probe.count() != calls+2 || deliveredName(t, probe) != "Second" {
		t.Errorf("%d Configure calls, last list %q, want %d and the later list", probe.count(), deliveredName(t, probe), calls+2)
	}
}

// An operation that runs while the plugin starts, before the Configure of its activation reads the
// list, leaves the plugin one Configure of the activation with the new list.
func TestAnOperationBeforeTheActivationConfigureEndsWithOneCallWithTheNewList(t *testing.T) {
	host, probe, source := newConfiguredHost(t)
	activate := prepareActivation(t, host, true)
	calls := probe.count()

	gate, entered := make(chan struct{}), make(chan struct{})
	operation := make(chan error, 1)
	go func() {
		operation <- host.ApplySecretChange(t.Context(), func() ([]string, error) {
			close(entered)
			<-gate
			source.rename("Renamed")
			return nil, nil
		})
	}()
	<-entered
	activation := make(chan error, 1)
	go func() { activation <- activate() }()
	plugintest.WaitUntil(t, "the activation waits for its Configure", func() bool {
		host.mu.RLock()
		defer host.mu.RUnlock()
		return host.awaitingFirstConfigure[secretPluginID]
	})
	close(gate)

	if err := <-operation; err != nil {
		t.Fatal(err)
	}
	if err := <-activation; err != nil {
		t.Fatal(err)
	}
	if probe.count() != calls+1 || deliveredName(t, probe) != "Renamed" {
		t.Errorf("%d Configure calls, last list %q, want %d and the new list", probe.count(), deliveredName(t, probe), calls+1)
	}
}

// An operation that runs while the Configure of the activation is on its way waits for it, and then
// sends the new list, so the plugin ends with it.
func TestAnOperationDuringTheActivationConfigureEndsWithTheNewListInThePlugin(t *testing.T) {
	host, probe, source := newConfiguredHost(t)
	activate := prepareActivation(t, host, true)
	calls := probe.count()
	entered, release := probe.setHold(t)

	activation := make(chan error, 1)
	go func() { activation <- activate() }()
	<-entered

	var ran atomic.Bool
	operation := make(chan error, 1)
	go func() {
		operation <- host.ApplySecretChange(t.Context(), func() ([]string, error) {
			ran.Store(true)
			source.rename("Renamed")
			return nil, nil
		})
	}()
	time.Sleep(150 * time.Millisecond)
	if ran.Load() {
		t.Error("the operation ran while the Configure of the activation was on its way")
	}
	release()

	if err := <-activation; err != nil {
		t.Fatal(err)
	}
	if err := <-operation; err != nil {
		t.Fatal(err)
	}
	if probe.count() != calls+2 || deliveredName(t, probe) != "Renamed" {
		t.Errorf("%d Configure calls, last list %q, want %d (the activation, then the operation) and the new list",
			probe.count(), deliveredName(t, probe), calls+2)
	}
}

// Renaming the group of a secret changes the list a plugin is given, so the plugin is sent it once.
func TestRenamingTheGroupOfASecretSendsOneConfigure(t *testing.T) {
	host, probe, source := newConfiguredHost(t)
	calls := probe.count()

	err := host.ApplySecretChange(t.Context(), func() ([]string, error) {
		source.regroup("mistral")
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if probe.count() != calls+1 {
		t.Fatalf("%d Configure calls after the group was renamed, want %d", probe.count(), calls+1)
	}
	if got := listed(t, probe, declared); len(got) != 1 || got[0].GetGroup() != "mistral" {
		t.Errorf("delivered %v, want the secret in the group mistral", got)
	}
}

// Two plugins whose sends fail are both named by the error, and a plugin after a failing one is
// still sent its list.
func TestTheErrorNamesEveryPluginWhoseSendFailed(t *testing.T) {
	const second, third = "md.numen.agents.second", "md.numen.agents.third"
	host, first, source := newConfiguredHost(t)
	secondProbe := addSecretPlugin(t, host, source, second)
	thirdProbe := addSecretPlugin(t, host, source, third)
	failure := connect.NewError(connect.CodeInternal, errors.New("the plugin cannot take it"))
	first.setFailure(failure)
	secondProbe.setFailure(failure)
	if err := host.NotifyPluginConfig(t.Context(), second); err == nil {
		t.Fatal("the second plugin accepted the list")
	}
	if err := host.NotifyPluginConfig(t.Context(), third); err != nil {
		t.Fatal(err)
	}
	secondCalls, thirdCalls := secondProbe.count(), thirdProbe.count()

	err := host.ApplySecretChange(t.Context(), source.renameBy("Renamed"))
	if err == nil {
		t.Fatal("the operation returned nil although two plugins did not accept the list")
	}
	for _, id := range []string{secretPluginID, second} {
		if !strings.Contains(err.Error(), id) {
			t.Errorf("the error %q does not name %s", err, id)
		}
	}
	if strings.Contains(err.Error(), third) {
		t.Errorf("the error %q names the plugin that accepted the list", err)
	}
	if secondProbe.count() != secondCalls+1 || thirdProbe.count() != thirdCalls+1 {
		t.Errorf("the plugins after the failing one got %d and %d calls, want 1 each",
			secondProbe.count()-secondCalls, thirdProbe.count()-thirdCalls)
	}
}
