package plugin

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugintest"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	pluginport "github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

// configureTimeout is how long the host waits for one Configure call.
const configureTimeout = 3 * time.Second

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

// listing is a secret source with one secret allowed to the plugin, whose name changes between reads.
type listing struct {
	mu   sync.Mutex
	name string
}

func newListing(name string) *listing { return &listing{name: name} }

func (l *listing) rename(name string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.name = name
}

func (l *listing) source() pluginport.SecretSource {
	return pluginport.SecretSource{List: func() ([]pluginport.Secret, error) {
		l.mu.Lock()
		defer l.mu.Unlock()
		return []pluginport.Secret{{
			ID: secretID, Name: l.name, Group: "openrouter",
			Allowed: []pluginport.SecretPermission{{Plugin: secretPluginID, Name: declared}},
		}}, nil
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
	return func() error {
		_, err := host.runActivation(t.Context(), p, activating)
		return err
	}
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

// The signature of a list whose Configure failed is not recorded as delivered, so the next refresh
// sends the list again.
func TestAFailedConfigureIsSentAgainByTheNextRefresh(t *testing.T) {
	host, probe, source := newConfiguredHost(t)
	probe.setFailure(connect.NewError(connect.CodeInternal, errors.New("the plugin cannot take it")))
	if err := host.ApplySecretChange(t.Context(), source.renameBy("Renamed")); err == nil {
		t.Fatal("the operation returned nil although the plugin did not accept the list")
	}
	probe.setFailure(nil)
	calls := probe.count()

	if err := host.RefreshSecrets(t.Context()); err != nil {
		t.Fatal(err)
	}
	if probe.count() != calls+1 || deliveredName(t, probe) != "Renamed" {
		t.Errorf("%d Configure calls after the refresh, want %d with the list that failed", probe.count(), calls+1)
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
// operation returns an error that names it.
func TestAHungConfigureDelaysTheOperationByTheTimeoutOnce(t *testing.T) {
	host, probe, source := newConfiguredHost(t)
	entered, _ := probe.setHold(t)

	start := time.Now()
	err := host.ApplySecretChange(t.Context(), source.renameBy("Renamed"))
	elapsed := time.Since(start)

	if len(entered) == 0 {
		t.Error("the plugin never received the call")
	}
	if elapsed < configureTimeout-500*time.Millisecond || elapsed > configureTimeout+1500*time.Millisecond {
		t.Errorf("the operation took %v, want about the Configure timeout of %v", elapsed, configureTimeout)
	}
	if err == nil || !strings.Contains(err.Error(), secretPluginID) {
		t.Errorf("the error = %v, want one that names the plugin", err)
	}
}

// A Configure of a plugin that never answers takes the timeout once, not twice.
func TestNotifyingAHungPluginWaitsForTheTimeoutOnce(t *testing.T) {
	host, probe, _ := newConfiguredHost(t)
	probe.setHold(t)

	start := time.Now()
	err := host.NotifyPluginConfig(t.Context(), secretPluginID)
	elapsed := time.Since(start)

	if elapsed > configureTimeout+1500*time.Millisecond {
		t.Errorf("the call took %v, want about the Configure timeout of %v", elapsed, configureTimeout)
	}
	if err == nil || !strings.Contains(err.Error(), secretPluginID) {
		t.Errorf("the error = %v, want one that names the plugin", err)
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
