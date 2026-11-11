package plugin_test

import (
	"context"
	"errors"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/delivery"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// collectFailures records the failures the host passes on.
type collectFailures struct {
	mu       sync.Mutex
	failures []plugindomain.Failure
}

func (c *collectFailures) add(failure plugindomain.Failure) {
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

func (l *listing) regroup(group string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.group = group
}

func (l *listing) allow(pluginID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.plugins = append(l.plugins, pluginID)
}

func (l *listing) source() plugindomain.SecretSource {
	return plugindomain.SecretSource{List: func() ([]plugindomain.Secret, error) {
		l.mu.Lock()
		defer l.mu.Unlock()
		var allowed []plugindomain.DeclarationKey
		for _, id := range l.plugins {
			allowed = append(allowed, plugindomain.DeclarationKey{Plugin: id, Name: declared})
		}
		return []plugindomain.Secret{{ID: secretID, Name: l.name, Group: l.group, Allowed: allowed}}, nil
	}}
}

func (l *listing) renameBy(name string) func() ([]string, error) {
	return func() ([]string, error) {
		l.rename(name)
		return nil, nil
	}
}

func deliveredName(t *testing.T, probe *configured) string {
	t.Helper()
	got := listed(t, probe, declared)
	if len(got) != 1 {
		t.Fatalf("delivered %v, want one secret", got)
	}
	return got[0].GetName()
}

func newConfiguredHost(t *testing.T) (*pluginhost.Host, *configured, *listing) {
	t.Helper()
	host, probe := newSecretHost(t, newKeys(map[string]string{"secret/" + secretID: fromKeychain}))
	source := newListing("Work")
	host.Config.SetSecretSource(source.source())
	if err := host.Config.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}
	return host, probe, source
}

func prepareActivation(t *testing.T, host *pluginhost.Host, dormant bool) func() error {
	t.Helper()
	activate := prepareActivationWith(t, host, dormant)
	return func() error { return activate(t.Context()) }
}

func prepareActivationWith(t *testing.T, host *pluginhost.Host, dormant bool) func(ctx context.Context) error {
	t.Helper()
	port := endpointOf(host, secretPluginID).Port
	sleeper := wiretest.StartSleepProcess(t).Process
	host.Processes.SetLauncher(func(context.Context, manifest.Discovered, int, string) (*os.Process, io.ReadCloser, error) {
		return sleeper, wiretest.GetReadyOutput(port), nil
	})
	p := getRawPlugin(host, secretPluginID)
	p.Manifest.Backend = manifest.Backend{Entrypoint: "bin/plugin"}
	activating := lifecycle.New(nil)
	activating.SetActivating()
	if dormant {
		dropEndpoint(host, secretPluginID)
		putSupervisor(host, secretPluginID, activating)
	}
	return func(ctx context.Context) error {
		_, err := plugin.RunActivation(ctx, host.Activation, p, activating)
		return err
	}
}

func addSecretPlugin(t *testing.T, host *pluginhost.Host, source *listing, pluginID string) *configured {
	t.Helper()
	probe := &configured{}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(probe))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	setPlugin(host, pluginID, manifest.Discovered{Manifest: manifest.Manifest{
		ID:      pluginID,
		Secrets: []manifest.SecretDeclaration{{Name: declared, Title: "API key"}},
	}})
	sup := lifecycle.New(nil)
	sup.ActivateInProcess(testsupport.GetMachine().Now())
	putSupervisor(host, pluginID, sup)
	putEndpoint(host, pluginID, transport.Endpoint{Host: "127.0.0.1", Port: wiretest.GetPort(t, server), Token: "second"})
	source.allow(pluginID)
	return probe
}

func TestFailedConfigureKeepsChangeAndRedactsValue(t *testing.T) {
	host, probe, source := newConfiguredHost(t)
	failed := &collectFailures{}
	host.Failures.SetOnPluginFailed(failed.add)
	probe.setFailure(connect.NewError(connect.CodeInternal, errors.New("rejected the key "+fromKeychain)))

	committed := false
	err := applyChange(host, t.Context(), func() ([]string, error) {
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
	sup := supervisorOf(host, secretPluginID)
	if sup.Failure.IsFailed() {
		t.Error("the plugin is marked as failed")
	}
	for _, info := range host.Registry.GetPluginInfos() {
		if info.ID == secretPluginID && info.Failure != nil {
			t.Errorf("the plugin page shows %+v", info.Failure)
		}
	}
}

func TestRefreshSkipsSameRejectedSecretList(t *testing.T) {
	host, probe, source := newConfiguredHost(t)
	reported := &collectErrors{}
	host.Failures.SetErrorHandler(reported.add)
	probe.setFailure(connect.NewError(connect.CodeInternal, errors.New("the plugin cannot take it")))
	if err := applyChange(host, t.Context(), source.renameBy("Renamed")); err == nil {
		t.Fatal("the operation returned nil although the plugin did not accept the list")
	}
	probe.setFailure(nil)
	calls := probe.count()

	if err := host.Config.Refresh(t.Context()); err != nil {
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
func TestRefresh_SendsToFailedPluginWhenListChanged(t *testing.T) {
	host, probe, source := newConfiguredHost(t)
	probe.setFailure(connect.NewError(connect.CodeInternal, errors.New("the plugin cannot take it")))
	if err := applyChange(host, t.Context(), source.renameBy("Renamed")); err == nil {
		t.Fatal("the operation returned nil although the plugin did not accept the list")
	}
	probe.setFailure(nil)
	calls := probe.count()
	source.rename("Renamed again")

	if err := host.Config.Refresh(t.Context()); err != nil {
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
	if err := applyChange(host, t.Context(), source.renameBy("Renamed")); err == nil {
		t.Fatal("the operation returned nil although the plugin did not accept the list")
	}
	probe.setFailure(nil)
	calls := probe.count()

	err := applyChange(host, t.Context(), func() ([]string, error) { return []string{secretPluginID}, nil })
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
	if err := applyChange(host, t.Context(), source.renameBy("Renamed")); err == nil {
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
	if err := host.Config.Refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	if probe.count() != calls+1 {
		t.Errorf("%d Configure calls after a refresh, want %d", probe.count(), calls+1)
	}
}

func TestFailedActivationConfigureReportedOnce(t *testing.T) {
	host, probe, source := newConfiguredHost(t)
	reported := &collectErrors{}
	host.Failures.SetErrorHandler(reported.add)
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
	if err := host.Config.Refresh(t.Context()); err != nil {
		t.Errorf("the refresh returned %v", err)
	}
	if probe.count() != calls+1 || reported.count() != 1 {
		t.Errorf("%d Configure calls and %d failures reported after a bare refresh, want %d and 1",
			probe.count(), reported.count(), calls+1)
	}

	source.rename("Renamed")
	if err := host.Config.Refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	if probe.count() != calls+2 || deliveredName(t, probe) != "Renamed" {
		t.Errorf("%d Configure calls, last list %q, want %d and the list that changed",
			probe.count(), deliveredName(t, probe), calls+2)
	}
}

func TestOperationUpdatingFailedPluginListSendsConfigure(t *testing.T) {
	host, probe, source := newConfiguredHost(t)
	probe.setFailure(connect.NewError(connect.CodeInternal, errors.New("the plugin cannot take it")))
	if err := applyChange(host, t.Context(), source.renameBy("Renamed")); err == nil {
		t.Fatal("the operation returned nil although the plugin did not accept the list")
	}
	probe.setFailure(nil)
	calls := probe.count()

	if err := applyChange(host, t.Context(), source.renameBy("Renamed again")); err != nil {
		t.Fatal(err)
	}
	if probe.count() != calls+1 || deliveredName(t, probe) != "Renamed again" {
		t.Errorf("%d Configure calls, last list %q, want %d and the list of the second operation",
			probe.count(), deliveredName(t, probe), calls+1)
	}
}

func TestUnrelatedOperationSkipsFailedPlugin(t *testing.T) {
	host, probe, source := newConfiguredHost(t)
	probe.setFailure(connect.NewError(connect.CodeInternal, errors.New("the plugin cannot take it")))
	if err := applyChange(host, t.Context(), source.renameBy("Renamed")); err == nil {
		t.Fatal("the operation returned nil although the plugin did not accept the list")
	}
	calls := probe.count()

	if err := applyChange(host, t.Context(), func() ([]string, error) { return nil, nil }); err != nil {
		t.Errorf("an operation that does not concern the plugin returned %v", err)
	}
	if probe.count() != calls {
		t.Errorf("%d Configure calls, want %d: the plugin was sent to by an operation that does not concern it", probe.count(), calls)
	}
}

func TestHungConfigureTimesOutOnce(t *testing.T) {
	host, probe, source := newConfiguredHost(t)
	entered, _ := probe.setHold(t)

	result := make(chan error, 1)
	ctx := t.Context()
	go func() { result <- applyChange(host, ctx, source.renameBy("Renamed")) }()
	select {
	case early := <-result:
		t.Fatalf("the operation returned %v before the Configure timeout", early)
	case <-time.After(delivery.ConfigureTimeout / 2):
	}
	var err error
	select {
	case err = <-result:
	case <-time.After(delivery.ConfigureTimeout + 2*time.Second):
		t.Fatal("the operation waits longer than the Configure timeout")
	}
	if len(entered) != 1 {
		t.Errorf("the plugin received %d calls, want 1", len(entered))
	}
	if err == nil || !strings.Contains(err.Error(), secretPluginID) || !strings.Contains(err.Error(), "timed out") {
		t.Errorf("the error = %v, want one that names the plugin and says the call timed out", err)
	}

	followUps := map[string]func() error{
		"the refresh after the operation": func() error { return host.Config.Refresh(ctx) },
		"an operation that concerns no plugin": func() error {
			return applyChange(host, ctx, func() ([]string, error) { return nil, nil })
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
		case <-time.After(delivery.ConfigureTimeout / 2):
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
	go func() { result <- host.Config.NotifyPluginConfig(ctx, secretPluginID) }()
	var err error
	select {
	case err = <-result:
	case <-time.After(delivery.ConfigureTimeout + 2*time.Second):
		t.Fatal("the call waits longer than the Configure timeout")
	}
	if err == nil || !strings.Contains(err.Error(), secretPluginID) || !strings.Contains(err.Error(), "timed out") {
		t.Errorf("the error = %v, want one that names the plugin and says the call timed out", err)
	}
}

func TestOperationWaitsForConfigureResponse(t *testing.T) {
	host, probe, source := newConfiguredHost(t)
	probe.mu.Lock()
	probe.delay = 300 * time.Millisecond
	probe.mu.Unlock()
	calls := probe.count()

	if err := applyChange(host, t.Context(), source.renameBy("Renamed")); err != nil {
		t.Fatal(err)
	}
	if probe.count() != calls+1 || deliveredName(t, probe) != "Renamed" {
		t.Errorf("%d Configure calls when the operation returned, want %d with the new list", probe.count(), calls+1)
	}
}

func TestConfigureCallbackDoesNotDeadlockRefresh(t *testing.T) {
	host, probe, source := newConfiguredHost(t)
	source.rename("Renamed")
	nested := make(chan error, 1)
	probe.mu.Lock()
	probe.during = func() {
		ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
		defer cancel()
		nested <- host.Config.Refresh(ctx)
	}
	probe.mu.Unlock()

	result := make(chan error, 1)
	go func() { result <- host.Config.Refresh(t.Context()) }()
	select {
	case err := <-result:
		if err != nil {
			t.Errorf("the round returned %v", err)
		}
	case <-time.After(delivery.ConfigureTimeout + 2*time.Second):
		t.Fatal("the round is stuck behind a handler that called back")
	}
	if err := <-nested; !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("the nested refresh returned %v, want the end of its own context", err)
	}
	probe.mu.Lock()
	probe.during = nil
	probe.mu.Unlock()
	source.rename("Again")
	if err := host.Config.Refresh(t.Context()); err != nil {
		t.Fatalf("the lock stays held after the round: %v", err)
	}
	if got := deliveredName(t, probe); got != "Again" {
		t.Errorf("the plugin holds %q, want the list of the refresh after the round", got)
	}
}

func TestRefreshBeforeActivationConfigureSendsLatestListOnce(t *testing.T) {
	host, probe, source := newConfiguredHost(t)
	activate := prepareActivation(t, host, true)
	calls := probe.count()
	if err := plugin.GetDelivery(host.Config).Lock(t.Context()); err != nil {
		t.Fatal(err)
	}
	activation := make(chan error, 1)
	go func() { activation <- activate() }()
	wiretest.WaitUntil(t, "the activation waits for its Configure", func() bool {
		return plugin.GetDelivery(host.Config).IsAwaitingFirstConfigure(secretPluginID)
	})
	source.rename("Renamed")
	refresh := make(chan error, 1)
	go func() { refresh <- host.Config.Refresh(t.Context()) }()
	plugin.GetDelivery(host.Config).Unlock()

	if err := <-activation; err != nil {
		t.Fatal(err)
	}
	if err := <-refresh; err != nil {
		t.Fatal(err)
	}
	if probe.count() != calls+1 || deliveredName(t, probe) != "Renamed" {
		t.Errorf("%d Configure calls, last list %q, want %d and the new list", probe.count(), deliveredName(t, probe), calls+1)
	}
}

func TestRefreshDuringActivationConfigureWaitsThenSendsLatestList(t *testing.T) {
	host, probe, source := newConfiguredHost(t)
	activate := prepareActivation(t, host, true)
	calls := probe.count()
	entered, release := probe.setHold(t)

	activation := make(chan error, 1)
	go func() { activation <- activate() }()
	<-entered

	source.rename("Renamed")
	refresh := make(chan error, 1)
	go func() { refresh <- host.Config.Refresh(t.Context()) }()
	select {
	case err := <-refresh:
		t.Fatalf("the refresh ran while the Configure of the activation was on its way: %v", err)
	case <-time.After(150 * time.Millisecond):
	}
	release()

	if err := <-activation; err != nil {
		t.Fatal(err)
	}
	if err := <-refresh; err != nil {
		t.Fatal(err)
	}
	if probe.count() != calls+2 || deliveredName(t, probe) != "Renamed" {
		t.Errorf("%d Configure calls, last list %q, want %d (the activation, then the refresh) and the new list",
			probe.count(), deliveredName(t, probe), calls+2)
	}
}

// Renaming the group of a secret changes the list a plugin is given, so the plugin is sent it once.
func TestRenamingTheGroupOfASecretSendsOneConfigure(t *testing.T) {
	host, probe, source := newConfiguredHost(t)
	calls := probe.count()

	err := applyChange(host, t.Context(), func() ([]string, error) {
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
	if err := host.Config.NotifyPluginConfig(t.Context(), second); err == nil {
		t.Fatal("the second plugin accepted the list")
	}
	if err := host.Config.NotifyPluginConfig(t.Context(), third); err != nil {
		t.Fatal(err)
	}
	secondCalls, thirdCalls := secondProbe.count(), thirdProbe.count()

	err := applyChange(host, t.Context(), source.renameBy("Renamed"))
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
