package plugin_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginhealth"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const configuredPluginID = "md.numen.agents.numen"

// refusedModel is the model the probe refuses its configuration for, with refusalText.
const (
	refusedModel = "retired-model"
	refusalText  = "Choose a model that is still offered"
)

// configured records the Configure calls a plugin receives.
type configured struct {
	wiretest.PluginLifecycle
	mu        sync.Mutex
	calls     []*v1.ConfigureRequest
	shutdowns int
}

func (c *configured) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.calls)
}

func (c *configured) Configure(
	_ context.Context, req *connect.Request[v1.ConfigureRequest],
) (*connect.Response[v1.ConfigureResponse], error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls = append(c.calls, req.Msg)
	if req.Msg.GetValues()["model"] == refusedModel {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New(refusalText))
	}
	return connect.NewResponse(&v1.ConfigureResponse{}), nil
}

// Shutdown counts the shutdown requests the plugin receives.
func (c *configured) Shutdown(
	_ context.Context, _ *connect.Request[v1.ShutdownRequest],
) (*connect.Response[v1.ShutdownResponse], error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.shutdowns++
	return connect.NewResponse(&v1.ShutdownResponse{}), nil
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

func newConfiguredHost(t *testing.T) (*pluginhost.Host, *configured) {
	t.Helper()
	probe := &configured{}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(probe))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	host := pluginhost.New(testsupport.GetMachine(), "desktop.editor")
	t.Cleanup(func() {
		if err := host.Management.Teardown.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond); err != nil {
			t.Error(err)
		}
	})
	setPlugin(host.Management, configuredPluginID, manifest.DiscoveredPlugin{Manifest: manifest.Manifest{
		ID: configuredPluginID,
		Settings: []manifest.Setting{
			{Key: "model", Type: "string", Title: "Model", Default: "sonnet"},
		},
	}})
	sup := lifecycle.New(nil)
	sup.ActivateInProcess(lifecycle.Lease{Kind: lifecycle.UseLease, Holder: getTestHolder()}, testsupport.GetMachine().Now())
	putSupervisor(host.Management, configuredPluginID, sup)
	putEndpoint(host.Management, configuredPluginID, transport.Endpoint{
		Host: "127.0.0.1", Port: wiretest.GetPort(t, server), Token: "launch-name",
	})
	return host, probe
}

func TestAPluginReceivesConfigureWithItsSettings(t *testing.T) {
	host, probe := newConfiguredHost(t)

	if err := host.Management.Config.NotifyPluginConfig(t.Context(), configuredPluginID); err != nil {
		t.Fatal(err)
	}
	if got := probe.last(t).GetValues()["model"]; got != "sonnet" {
		t.Errorf("model = %q, want the default of the model setting", got)
	}
}

func TestARefreshConfiguresAPluginOnlyWhenForced(t *testing.T) {
	host, probe := newConfiguredHost(t)
	if err := host.Management.Config.NotifyPluginConfig(t.Context(), configuredPluginID); err != nil {
		t.Fatal(err)
	}
	calls := probe.count()

	if err := host.Management.Config.Refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	if probe.count() != calls {
		t.Errorf("%d Configure calls after a plain refresh, want %d", probe.count(), calls)
	}

	if err := host.Management.Config.Refresh(t.Context(), configuredPluginID); err != nil {
		t.Fatal(err)
	}
	if probe.count() != calls+1 {
		t.Errorf("%d Configure calls after a forced refresh, want %d", probe.count(), calls+1)
	}
}

func TestAChangeOfASettingReachesAPlugin(t *testing.T) {
	host, probe := newConfiguredHost(t)

	if err := updateSetting(t.Context(), host, configuredPluginID, "model", "opus"); err != nil {
		t.Fatal(err)
	}
	if got := probe.last(t).GetValues()["model"]; got != "opus" {
		t.Errorf("model = %q, want opus", got)
	}
}

func TestAPluginAwaitingItsActivationConfigureIsNotDueOne(t *testing.T) {
	host, probe := newConfiguredHost(t)
	plugin.GetDelivery(host.Management.Config).AwaitFirstConfigure(configuredPluginID)

	if err := host.Management.Config.Refresh(t.Context(), configuredPluginID); err != nil {
		t.Fatal(err)
	}
	if probe.count() != 0 {
		t.Errorf("the refresh sent %d Configure calls to a plugin that awaits its activation Configure", probe.count())
	}
}

func getHealthItem(supervision *container.PluginSupervision, pluginID, key string) *plugindomain.HealthItem {
	for _, item := range supervision.Health.List() {
		if item.PluginID == pluginID && item.Key == key {
			return &item
		}
	}
	return nil
}

// A Configure the plugin refuses is listed as its configure item with the plugin's message, and the next
// Configure it takes clears the item.
func TestARefusedConfigureIsTheConfigureItemUntilAConfigureIsTaken(t *testing.T) {
	host, _ := newConfiguredHost(t)

	if err := updateSetting(t.Context(), host, configuredPluginID, "model", refusedModel); err == nil {
		t.Fatal("a refused Configure answered no error")
	}
	item := getHealthItem(host.Supervision, configuredPluginID, plugindomain.HealthKeyConfigure)
	if item == nil || item.Severity != plugindomain.HealthError || item.Text != refusalText {
		t.Fatalf("configure item = %+v, want an error with the message of the plugin", item)
	}
	if item.Action == nil || item.Action.Command != plugindomain.AppCommandOpenSettings || item.Action.Values[plugindomain.OpenSettingsPlugin] != configuredPluginID {
		t.Fatalf("action = %+v, want the settings of the plugin", item.Action)
	}

	if err := updateSetting(t.Context(), host, configuredPluginID, "model", "opus"); err != nil {
		t.Fatal(err)
	}
	if item := getHealthItem(host.Supervision, configuredPluginID, plugindomain.HealthKeyConfigure); item != nil {
		t.Errorf("configure item after a taken Configure = %+v, want none", item)
	}
}

// The items a plugin reports stay while it is stopped as idle, and go once it is switched off.
func TestTheItemsOfAPluginOutliveAnIdleStopAndGoWhenItIsSwitchedOff(t *testing.T) {
	host, _ := newConfiguredHost(t)
	report := pluginhealth.NewReport(host.Supervision.Health, host.Management.Registry)
	if err := report.Execute(configuredPluginID, plugindomain.HealthItem{Key: "connection", Severity: plugindomain.HealthError, Text: "Choose a connection"}); err != nil {
		t.Fatal(err)
	}

	host.Management.Teardown.EvictIdle(t.Context(), 0, time.Millisecond)
	if item := getHealthItem(host.Supervision, configuredPluginID, "connection"); item == nil {
		t.Fatal("an idle stop took the item of the plugin out")
	}

	if err := setEnabled(t.Context(), host, configuredPluginID, false); err != nil {
		t.Fatal(err)
	}
	if items := host.Supervision.Health.List(); len(items) != 0 {
		t.Errorf("items of a plugin switched off = %+v, want none", items)
	}
}

func TestSafeModeRemovesTheItemsOfEveryPlugin(t *testing.T) {
	host, _ := newConfiguredHost(t)
	slow := plugindomain.HealthItem{Key: "connection", Severity: plugindomain.HealthWarning, Text: "The connection is slow"}
	if err := pluginhealth.NewReport(host.Supervision.Health, host.Management.Registry).Execute(configuredPluginID, slow); err != nil {
		t.Fatal(err)
	}

	host.Management.Teardown.SetSafeMode(true)

	if items := host.Supervision.Health.List(); len(items) != 0 {
		t.Errorf("items in safe mode = %+v, want none", items)
	}
}
