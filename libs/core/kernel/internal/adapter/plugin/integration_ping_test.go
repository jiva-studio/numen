package plugin_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func launchAnsweringPing(t *testing.T, protocol int32, procs *[]*os.Process) supervision.Launcher {
	t.Helper()
	pingPort := wiretest.ServePing(t, protocol)
	return func(ctx context.Context, _ manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		cmd := wiretest.NewIdleCommand(context.WithoutCancel(ctx))
		if err := cmd.Start(); err != nil {
			return nil, nil, err
		}
		t.Cleanup(func() { wiretest.KillProcess(t, cmd.Process) })
		*procs = append(*procs, cmd.Process)
		return cmd.Process, io.NopCloser(strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", pingPort))), nil
	}
}

func TestEnsureActive_RefusesPluginWithMismatchedProtocol(t *testing.T) {
	cases := []struct {
		name     string
		protocol int32
		want     string
	}{
		{"a newer protocol", manifest.ProtocolVersion + 1, "uses protocol 2, its manifest declares 1"},
		{"no protocol reported", 0, "uses protocol 0, its manifest declares 1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			host := newDiscoveredHost(t)
			var failedID string
			var reason string
			host.Failures.SetOnPluginFailed(func(failed plugindomain.Failure) {
				failedID, reason = failed.PluginID, failed.Reason
			})
			var procs []*os.Process
			host.Processes.SetLauncher(launchAnsweringPing(t, tc.protocol, &procs))

			_, err := host.Activation.EnsureActive(t.Context(), "org.example.sample")
			if !errors.Is(err, manifest.ErrUnsupportedProtocol) || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("EnsureActive() = %v, want %v naming %q", err, manifest.ErrUnsupportedProtocol, tc.want)
			}
			if failedID != "org.example.sample" || reason != lifecycle.DescribeFailure(manifest.ErrUnsupportedProtocol) {
				t.Fatalf("failure reported for %q with %q, want org.example.sample with the protocol reason", failedID, reason)
			}
			if err := procs[0].Signal(syscall.Signal(0)); !errors.Is(err, os.ErrProcessDone) {
				t.Fatalf("signal the rejected process: %v, want %v", err, os.ErrProcessDone)
			}
			if _, err := host.Activation.EnsureActive(t.Context(), "org.example.sample"); !errors.Is(err, manifest.ErrUnsupportedProtocol) {
				t.Fatalf("second EnsureActive() = %v, want a new start that fails with %v", err, manifest.ErrUnsupportedProtocol)
			}
			if len(procs) != 2 {
				t.Fatalf("launched %d processes, want one for each use", len(procs))
			}
		})
	}
}

func TestEnsureActive_ActivatesPluginWithHostProtocol(t *testing.T) {
	host := newDiscoveredHost(t)
	var procs []*os.Process
	host.Processes.SetLauncher(launchAnsweringPing(t, manifest.ProtocolVersion, &procs))

	if _, err := host.Activation.EnsureActive(t.Context(), "org.example.sample"); err != nil {
		t.Fatalf("EnsureActive() = %v, want the plugin active", err)
	}
	host.Activation.Release("org.example.sample")
	if err := host.Teardown.TeardownAll(t.Context(), 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

func TestCheckPluginProtocol_RejectsUnsupportedProtocol(t *testing.T) {
	newer := manifest.ProtocolVersion + 1
	endpoint := transport.Endpoint{Host: "127.0.0.1", Port: wiretest.ServePing(t, int32(newer)), Token: ""}
	declared := manifest.Manifest{ID: "md.numen.newer", Protocol: newer}

	err := transport.CheckPluginProtocol(t.Context(), transport.NewClients().Get(endpoint), declared, endpoint)
	if !errors.Is(err, manifest.ErrUnsupportedProtocol) || !strings.Contains(err.Error(), "this host uses 1") {
		t.Fatalf("checkPluginProtocol() = %v, want %v naming the host's protocol", err, manifest.ErrUnsupportedProtocol)
	}
}

func TestCheckPluginProtocolNamesAPluginThatDoesNotAnswerPing(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	addr, ok := srv.Listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("expected *net.TCPAddr, got %T", srv.Listener.Addr())
	}
	endpoint := transport.Endpoint{Host: "127.0.0.1", Port: addr.Port, Token: ""}

	err := transport.CheckPluginProtocol(t.Context(), transport.NewClients().Get(endpoint), manifest.Manifest{ID: "md.numen.silent"}, endpoint)
	if err == nil || !strings.Contains(err.Error(), "ping plugin md.numen.silent") {
		t.Fatalf("checkPluginProtocol() = %v, want the ping failure naming the plugin", err)
	}
}
