package integration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugindir"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugintest"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// launchAnsweringPing returns a launcher whose process hands over the port of a server answering Ping with protocol.
func launchAnsweringPing(t *testing.T, protocol int32, procs *[]*os.Process) plugin.Launcher {
	t.Helper()
	pingPort := plugintest.ServePing(t, protocol)
	return func(_ context.Context, _ plugindir.DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		cmd := plugintest.NewIdleCommand()
		if err := cmd.Start(); err != nil {
			return nil, nil, err
		}
		t.Cleanup(func() { plugintest.KillProcess(t, cmd.Process) })
		*procs = append(*procs, cmd.Process)
		return cmd.Process, io.NopCloser(strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", pingPort))), nil
	}
}

func TestAPluginThatSpeaksAnotherProtocolIsRefusedAndFailsAtOnce(t *testing.T) {
	cases := []struct {
		name     string
		protocol int32
		want     string
	}{
		{"a newer protocol", pluginport.PluginProtocolVersion + 1, "speaks protocol 2, its manifest declares 1"},
		{"no protocol reported", 0, "speaks protocol 0, its manifest declares 1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			host := newDiscoveredHost(t)
			var failedID string
			var reason string
			host.SetOnPluginFailed(func(failed pluginport.PluginFailure) {
				failedID, reason = failed.PluginID, failed.Reason
			})
			var procs []*os.Process
			host.SetLauncher(launchAnsweringPing(t, tc.protocol, &procs))

			_, err := host.EnsureActive(t.Context(), "org.example.sample")
			if !errors.Is(err, pluginport.ErrUnsupportedProtocol) || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("EnsureActive() = %v, want %v naming %q", err, pluginport.ErrUnsupportedProtocol, tc.want)
			}
			if failedID != "org.example.sample" || reason != lifecycle.DescribeFailure(pluginport.ErrUnsupportedProtocol) {
				t.Fatalf("failure reported for %q with %q, want org.example.sample with the protocol reason", failedID, reason)
			}
			if err := procs[0].Signal(syscall.Signal(0)); !errors.Is(err, os.ErrProcessDone) {
				t.Fatalf("signal the rejected process: %v, want %v", err, os.ErrProcessDone)
			}
			if _, err := host.EnsureActive(t.Context(), "org.example.sample"); !errors.Is(err, lifecycle.ErrPluginFailed) {
				t.Fatalf("second EnsureActive() = %v, want %v", err, lifecycle.ErrPluginFailed)
			}
			if len(procs) != 1 {
				t.Fatalf("launched %d processes, want 1", len(procs))
			}
		})
	}
}

func TestAPluginThatSpeaksTheHostsProtocolIsActivated(t *testing.T) {
	host := newDiscoveredHost(t)
	var procs []*os.Process
	host.SetLauncher(launchAnsweringPing(t, pluginport.PluginProtocolVersion, &procs))

	if _, err := host.EnsureActive(t.Context(), "org.example.sample"); err != nil {
		t.Fatalf("EnsureActive() = %v, want the plugin active", err)
	}
	host.Release("org.example.sample")
	if err := host.TeardownAll(t.Context(), 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

func TestCheckPluginProtocolRefusesAProtocolTheHostDoesNotSpeak(t *testing.T) {
	newer := pluginport.PluginProtocolVersion + 1
	endpoint := transport.Endpoint{Host: "127.0.0.1", Port: plugintest.ServePing(t, int32(newer)), Token: ""}
	manifest := plugindir.Manifest{ID: "md.numen.newer", Protocol: newer}

	err := transport.CheckPluginProtocol(t.Context(), manifest, endpoint)
	if !errors.Is(err, pluginport.ErrUnsupportedProtocol) || !strings.Contains(err.Error(), "this host speaks 1") {
		t.Fatalf("checkPluginProtocol() = %v, want %v naming the host's protocol", err, pluginport.ErrUnsupportedProtocol)
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

	err := transport.CheckPluginProtocol(t.Context(), plugindir.Manifest{ID: "md.numen.silent"}, endpoint)
	if err == nil || !strings.Contains(err.Error(), "ping plugin md.numen.silent") {
		t.Fatalf("checkPluginProtocol() = %v, want the ping failure naming the plugin", err)
	}
}
