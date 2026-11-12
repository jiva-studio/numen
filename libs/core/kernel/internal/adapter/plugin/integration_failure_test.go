package plugin_test

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const failingPluginID = "org.example.sample"

func failLaunching(_ context.Context, _ manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
	return nil, nil, errors.New("exec format error")
}

func getFailure(host *container.PluginRoles, id string) *plugindomain.HealthItem {
	for _, item := range host.Health.List() {
		if item.PluginID == id && item.Key == plugindomain.HealthKeyStart {
			return &item
		}
	}
	return nil
}

func TestAPluginWhoseStartFailedIsListedWithItsReason(t *testing.T) {
	host := newDiscoveredHost(t)
	var procs []*os.Process
	host.Processes.SetLauncher(launchAnsweringPing(t, manifest.ProtocolVersion+1, &procs))

	if got := getFailure(host, failingPluginID); got != nil {
		t.Fatalf("failure before any start = %+v, want none", got)
	}
	if _, err := host.Activation.EnsureActive(t.Context(), failingPluginID); !errors.Is(err, manifest.ErrUnsupportedProtocol) {
		t.Fatalf("EnsureActive() = %v, want %v", err, manifest.ErrUnsupportedProtocol)
	}

	got := getFailure(host, failingPluginID)
	if got == nil {
		t.Fatal("a plugin on an unsupported protocol is listed without a failure")
	}
	if got.Severity != plugindomain.HealthError || got.Detail == "" {
		t.Fatalf("failure = %+v, want an error with the cause as its detail", got)
	}
	if want := lifecycle.DescribeFailure(manifest.ErrUnsupportedProtocol); got.Text != want {
		t.Fatalf("reason = %q, want %q", got.Text, want)
	}
	if got.Action == nil || got.Action.Command != plugindomain.AppCommandOpenSettings || got.Action.Values[plugindomain.OpenSettingsPlugin] != failingPluginID {
		t.Fatalf("action = %+v, want the settings of the plugin", got.Action)
	}
}

func TestRetryStartsAFailedPluginAndTellsItRecovered(t *testing.T) {
	host := newDiscoveredHost(t)
	var procs []*os.Process
	host.Processes.SetLauncher(launchAnsweringPing(t, manifest.ProtocolVersion+1, &procs))
	var recovered []string
	host.Failures.SetOnFailureCleared(func(id string) { recovered = append(recovered, id) })
	if _, err := host.Activation.EnsureActive(t.Context(), failingPluginID); err == nil {
		t.Fatal("expected the first start to fail")
	}

	host.Processes.SetLauncher(launchAnsweringPing(t, manifest.ProtocolVersion, &procs))
	failure, err := host.Retry.RetryPlugin(t.Context(), failingPluginID)
	if err != nil || failure != nil {
		t.Fatalf("RetryPlugin() = %+v, %v, want the plugin started", failure, err)
	}

	if got := getFailure(host, failingPluginID); got != nil {
		t.Fatalf("failure after a good retry = %+v, want none", got)
	}
	if status, _ := host.Registry.GetPluginStatus(failingPluginID); !status.IsActive {
		t.Fatal("the retried plugin is not running")
	}
	if len(recovered) != 1 || recovered[0] != failingPluginID {
		t.Fatalf("recovered = %v, want one report of %s", recovered, failingPluginID)
	}
}

func TestARetryThatFailsAgainAnswersTheReason(t *testing.T) {
	host := newDiscoveredHost(t)
	host.Processes.SetLauncher(failLaunching)
	failure, err := host.Retry.RetryPlugin(t.Context(), failingPluginID)
	if err != nil {
		t.Fatalf("RetryPlugin() error = %v", err)
	}
	if failure == nil || failure.Reason != lifecycle.DescribeFailure(errors.New("launch plugin org.example.sample: exec format error")) {
		t.Fatalf("failure = %+v, want the reason the start failed for", failure)
	}
	if failure.Detail != "launch plugin org.example.sample: exec format error" {
		t.Fatalf("failure detail = %q, want the launch error", failure.Detail)
	}
	if got := getFailure(host, failingPluginID); got == nil || got.Text != failure.Reason || got.Detail != failure.Detail {
		t.Fatalf("listed failure = %+v, want %+v", got, failure)
	}
	if items := host.Health.List(); len(items) != 1 {
		t.Fatalf("health items = %+v, want the one start item", items)
	}
}

func TestTheReasonOfAFailedStartIsListedUntilAStartSucceeds(t *testing.T) {
	host := newDiscoveredHost(t)
	var procs []*os.Process
	host.Processes.SetLauncher(failLaunching)
	if _, err := host.Activation.EnsureActive(t.Context(), failingPluginID); err == nil {
		t.Fatal("expected the first start to fail")
	}
	if got := getFailure(host, failingPluginID); got == nil {
		t.Fatal("the failed start is listed without its reason")
	}
	var recovered []string
	host.Failures.SetOnFailureCleared(func(id string) { recovered = append(recovered, id) })

	host.Processes.SetLauncher(launchAnsweringPing(t, manifest.ProtocolVersion, &procs))
	if _, err := host.Activation.EnsureActive(t.Context(), failingPluginID); err != nil {
		t.Fatalf("EnsureActive() on the next use = %v, want the plugin started", err)
	}

	if got := getFailure(host, failingPluginID); got != nil {
		t.Fatalf("failure after a start that succeeded = %+v, want none", got)
	}
	if len(recovered) != 1 || recovered[0] != failingPluginID {
		t.Fatalf("recovered = %v, want one report of %s", recovered, failingPluginID)
	}
}

func TestRetryOfAPluginTheHostDoesNotKnowIsNotInstalled(t *testing.T) {
	host := newDiscoveredHost(t)

	_, err := host.Retry.RetryPlugin(t.Context(), "md.numen.missing")

	var unavailable *plugindomain.UnavailableError
	if !errors.As(err, &unavailable) || unavailable.Reason != plugindomain.NotInstalled {
		t.Fatalf("RetryPlugin() = %v, want a plugin that is not installed", err)
	}
}

func TestEnablingAFailedPluginTellsItRecovered(t *testing.T) {
	host := newDiscoveredHost(t)
	host.Processes.SetLauncher(failLaunching)
	if _, err := host.Retry.RetryPlugin(t.Context(), failingPluginID); err != nil {
		t.Fatalf("RetryPlugin() error = %v", err)
	}
	var recovered []string
	host.Failures.SetOnFailureCleared(func(id string) { recovered = append(recovered, id) })

	if err := setEnabled(t.Context(), host, failingPluginID, true); err != nil {
		t.Fatalf("SetPluginEnabled() error = %v", err)
	}

	if len(recovered) != 1 || recovered[0] != failingPluginID {
		t.Fatalf("recovered = %v, want one report of %s", recovered, failingPluginID)
	}
	if got := getFailure(host, failingPluginID); got != nil {
		t.Fatalf("failure after enabling = %+v, want none", got)
	}
}

func TestAFailureIsDescribedInWordsAPersonReads(t *testing.T) {
	cases := []struct {
		name  string
		cause error
		want  string
	}{
		{"an unsupported protocol", manifest.ErrUnsupportedProtocol, "Unsupported plugin protocol. Update the plugin or Numen"},
		{"a handshake that never came", transport.ErrHandshakeTimeout, "Timed out while starting"},
		{"anything else", errors.New("exec format error"), "Failed to start"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := lifecycle.DescribeFailure(tc.cause); got != tc.want {
				t.Fatalf("describeFailure() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDisablingAFailedPluginClearsItsFailure(t *testing.T) {
	host := newDiscoveredHost(t)
	host.Processes.SetLauncher(failLaunching)
	if _, err := host.Retry.RetryPlugin(t.Context(), failingPluginID); err != nil {
		t.Fatalf("RetryPlugin() error = %v", err)
	}
	var cleared []string
	host.Failures.SetOnFailureCleared(func(id string) { cleared = append(cleared, id) })

	if err := setEnabled(t.Context(), host, failingPluginID, false); err != nil {
		t.Fatalf("SetPluginEnabled() error = %v", err)
	}

	if len(cleared) != 1 || cleared[0] != failingPluginID {
		t.Fatalf("cleared = %v, want one report of %s", cleared, failingPluginID)
	}
	if got := getFailure(host, failingPluginID); got != nil {
		t.Fatalf("failure after disabling = %+v, want none", got)
	}
}
