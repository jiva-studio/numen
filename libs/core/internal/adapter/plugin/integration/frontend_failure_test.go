package integration

import (
	"errors"
	"os"
	"syscall"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

const frontendLate = "the plugin did not start within 30 s"

func TestTheBackendOfAPluginWhoseFrontendDidNotStartIsStopped(t *testing.T) {
	host := newDiscoveredHost(t)
	var procs []*os.Process
	host.SetLauncher(launchAnsweringPing(t, pluginport.PluginProtocolVersion, &procs))
	if _, err := host.EnsureActive(t.Context(), failingPluginID); err != nil {
		t.Fatalf("EnsureActive() error = %v", err)
	}
	host.Release(failingPluginID)

	failure, err := host.StopFailedPlugin(t.Context(), failingPluginID, frontendLate)
	if err != nil {
		t.Fatalf("StopFailedPlugin() error = %v", err)
	}

	if status, _ := host.GetPluginStatus(failingPluginID); status.IsActive {
		t.Fatal("the backend still runs")
	}
	if len(procs) != 1 {
		t.Fatalf("launched %d processes, want 1", len(procs))
	}
	if err := procs[0].Signal(syscall.Signal(0)); !errors.Is(err, os.ErrProcessDone) {
		t.Fatalf("signal the stopped process: %v, want %v", err, os.ErrProcessDone)
	}
	if failure == nil || failure.Detail != frontendLate {
		t.Fatalf("failure = %+v, want the reason the frontend gave", failure)
	}
	if got := getFailure(t, host, failingPluginID); got == nil || got.Detail != frontendLate {
		t.Fatalf("listed failure = %+v, want %q", got, frontendLate)
	}
	if _, err := host.EnsureActive(t.Context(), failingPluginID); !errors.Is(err, lifecycle.ErrPluginFailed) {
		t.Fatalf("EnsureActive() after the stop = %v, want %v", err, lifecycle.ErrPluginFailed)
	}
}

func TestStoppingAPluginTheHostDoesNotKnowIsNotInstalled(t *testing.T) {
	host := newDiscoveredHost(t)

	_, err := host.StopFailedPlugin(t.Context(), "md.numen.missing", frontendLate)

	var unavailable *pluginport.UnavailableError
	if !errors.As(err, &unavailable) || unavailable.Reason != pluginport.PluginNotInstalled {
		t.Fatalf("StopFailedPlugin() = %v, want a plugin that is not installed", err)
	}
}
