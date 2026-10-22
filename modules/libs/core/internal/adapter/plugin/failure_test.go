package plugin

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

const failingPluginID = "md.numen.engines.ocr"

func failLaunching(_ context.Context, _ DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
	return nil, nil, errors.New("exec format error")
}

func getFailure(t *testing.T, host *Host, id string) *port.PluginFailure {
	t.Helper()
	for _, info := range host.GetPluginInfos() {
		if info.ID == id {
			return info.Failure
		}
	}
	t.Fatalf("plugin %s is not listed", id)
	return nil
}

func TestAPluginTheHostContainsOffIsListedWithItsReason(t *testing.T) {
	host := newDiscoveredHost(t)
	var procs []*os.Process
	host.SetLauncher(launchAnsweringPing(t, port.PluginProtocolVersion+1, &procs))

	if got := getFailure(t, host, failingPluginID); got != nil {
		t.Fatalf("failure before any start = %+v, want none", got)
	}
	if _, err := host.EnsureActive(t.Context(), failingPluginID); !errors.Is(err, port.ErrUnsupportedProtocol) {
		t.Fatalf("EnsureActive() = %v, want %v", err, port.ErrUnsupportedProtocol)
	}

	got := getFailure(t, host, failingPluginID)
	if got == nil {
		t.Fatal("a plugin on an unsupported protocol is listed without a failure")
	}
	if got.PluginID != failingPluginID || got.Name == "" || got.At.IsZero() {
		t.Fatalf("failure = %+v, want the id, the name and the time", got)
	}
	if want := describeFailure(port.ErrUnsupportedProtocol); got.Reason != want {
		t.Fatalf("reason = %q, want %q", got.Reason, want)
	}
}

func TestRetryStartsAFailedPluginAndTellsItRecovered(t *testing.T) {
	host := newDiscoveredHost(t)
	var procs []*os.Process
	host.SetLauncher(launchAnsweringPing(t, port.PluginProtocolVersion+1, &procs))
	var recovered []string
	host.SetOnFailureCleared(func(id string) { recovered = append(recovered, id) })
	if _, err := host.EnsureActive(t.Context(), failingPluginID); err == nil {
		t.Fatal("expected the first start to fail")
	}

	host.SetLauncher(launchAnsweringPing(t, port.PluginProtocolVersion, &procs))
	failure, err := host.RetryPlugin(t.Context(), failingPluginID)
	if err != nil || failure != nil {
		t.Fatalf("RetryPlugin() = %+v, %v, want the plugin started", failure, err)
	}

	if got := getFailure(t, host, failingPluginID); got != nil {
		t.Fatalf("failure after a good retry = %+v, want none", got)
	}
	if status, _ := host.GetPluginStatus(failingPluginID); !status.IsActive {
		t.Fatal("the retried plugin is not running")
	}
	if len(recovered) != 1 || recovered[0] != failingPluginID {
		t.Fatalf("recovered = %v, want one report of %s", recovered, failingPluginID)
	}
}

func TestARetryThatFailsAgainContainsThePluginOffAtOnce(t *testing.T) {
	host := newDiscoveredHost(t)
	host.SetLauncher(failLaunching)
	var notified []port.PluginFailure
	host.SetOnPluginFailed(func(failure port.PluginFailure) { notified = append(notified, failure) })

	failure, err := host.RetryPlugin(t.Context(), failingPluginID)
	if err != nil {
		t.Fatalf("RetryPlugin() error = %v", err)
	}
	if failure == nil || failure.Reason != describeFailure(errors.New("launch plugin md.numen.engines.ocr: exec format error")) {
		t.Fatalf("failure = %+v, want the reason the start failed for", failure)
	}
	if got := getFailure(t, host, failingPluginID); got == nil || got.Reason != failure.Reason {
		t.Fatalf("listed failure = %+v, want %+v", got, failure)
	}
	if len(notified) != 1 || notified[0].PluginID != failingPluginID {
		t.Fatalf("failure callbacks = %+v, want one for %s", notified, failingPluginID)
	}
}

func TestRetryOfAPluginTheHostDoesNotKnowIsNotInstalled(t *testing.T) {
	host := newDiscoveredHost(t)

	_, err := host.RetryPlugin(t.Context(), "md.numen.missing")

	var unavailable *port.UnavailableError
	if !errors.As(err, &unavailable) || unavailable.Reason != port.PluginNotInstalled {
		t.Fatalf("RetryPlugin() = %v, want a plugin that is not installed", err)
	}
}

func TestEnablingAFailedPluginTellsItRecovered(t *testing.T) {
	host := newDiscoveredHost(t)
	host.SetLauncher(failLaunching)
	if _, err := host.RetryPlugin(t.Context(), failingPluginID); err != nil {
		t.Fatalf("RetryPlugin() error = %v", err)
	}
	var recovered []string
	host.SetOnFailureCleared(func(id string) { recovered = append(recovered, id) })

	if err := host.SetPluginEnabled(t.Context(), failingPluginID, true); err != nil {
		t.Fatalf("SetPluginEnabled() error = %v", err)
	}

	if len(recovered) != 1 || recovered[0] != failingPluginID {
		t.Fatalf("recovered = %v, want one report of %s", recovered, failingPluginID)
	}
	if got := getFailure(t, host, failingPluginID); got != nil {
		t.Fatalf("failure after enabling = %+v, want none", got)
	}
}

func TestAFailureIsDescribedInWordsAPersonReads(t *testing.T) {
	cases := []struct {
		name  string
		cause error
		want  string
	}{
		{"an unsupported protocol", port.ErrUnsupportedProtocol, "The plugin speaks a plugin protocol version this Numen does not support. Update the plugin or Numen."},
		{"a handshake that never came", errHandshakeTimeout, "The plugin did not finish starting in time."},
		{"anything else", errors.New("exec format error"), "The plugin could not be started: exec format error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := describeFailure(tc.cause); got != tc.want {
				t.Fatalf("describeFailure() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDisablingAFailedPluginClearsItsFailure(t *testing.T) {
	host := newDiscoveredHost(t)
	host.SetLauncher(failLaunching)
	if _, err := host.RetryPlugin(t.Context(), failingPluginID); err != nil {
		t.Fatalf("RetryPlugin() error = %v", err)
	}
	var cleared []string
	host.SetOnFailureCleared(func(id string) { cleared = append(cleared, id) })

	if err := host.SetPluginEnabled(t.Context(), failingPluginID, false); err != nil {
		t.Fatalf("SetPluginEnabled() error = %v", err)
	}

	if len(cleared) != 1 || cleared[0] != failingPluginID {
		t.Fatalf("cleared = %v, want one report of %s", cleared, failingPluginID)
	}
	if got := getFailure(t, host, failingPluginID); got != nil {
		t.Fatalf("failure after disabling = %+v, want none", got)
	}
}
