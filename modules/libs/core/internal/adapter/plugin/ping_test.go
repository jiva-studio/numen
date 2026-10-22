package plugin

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

	"connectrpc.com/connect"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// protocolLifecycle answers Ping with the protocol it holds.
type protocolLifecycle struct {
	numenv1connect.UnimplementedPluginLifecycleServiceHandler
	protocol int32
}

func (p protocolLifecycle) Ping(
	_ context.Context,
	_ *connect.Request[v1.PingRequest],
) (*connect.Response[v1.PingResponse], error) {
	return connect.NewResponse(&v1.PingResponse{Status: "ok", Protocol: p.protocol}), nil
}

// servePing starts a server answering Ping with the given protocol and returns its port.
func servePing(t *testing.T, protocol int32) int {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(protocolLifecycle{protocol: protocol}))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	addr, ok := srv.Listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("expected *net.TCPAddr, got %T", srv.Listener.Addr())
	}
	return addr.Port
}

// serveHostPing starts a server answering Ping with the host's protocol and returns its port.
func serveHostPing(t *testing.T) int {
	t.Helper()
	return servePing(t, port.PluginProtocolVersion)
}

// launchAnsweringPing returns a launcher whose process hands over the port of a server answering Ping with protocol.
func launchAnsweringPing(t *testing.T, protocol int32, procs *[]*os.Process) Launcher {
	t.Helper()
	pingPort := servePing(t, protocol)
	return func(_ context.Context, _ DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		cmd := newIdleCommand()
		if err := cmd.Start(); err != nil {
			return nil, nil, err
		}
		t.Cleanup(func() { killProcess(t, cmd.Process) })
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
		{"a newer protocol", port.PluginProtocolVersion + 1, "speaks protocol 2, its manifest declares 1"},
		{"no protocol reported", 0, "speaks protocol 0, its manifest declares 1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			host := newDiscoveredHost(t)
			var failedID string
			var reason string
			host.SetOnPluginFailed(func(failed port.PluginFailure) {
				failedID, reason = failed.PluginID, failed.Reason
			})
			var procs []*os.Process
			host.SetLauncher(launchAnsweringPing(t, tc.protocol, &procs))

			_, err := host.EnsureActive(t.Context(), "md.numen.engines.ocr")
			if !errors.Is(err, port.ErrUnsupportedProtocol) || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("EnsureActive() = %v, want %v naming %q", err, port.ErrUnsupportedProtocol, tc.want)
			}
			if failedID != "md.numen.engines.ocr" || reason != describeFailure(port.ErrUnsupportedProtocol) {
				t.Fatalf("failure reported for %q with %q, want md.numen.engines.ocr with the protocol reason", failedID, reason)
			}
			if err := procs[0].Signal(syscall.Signal(0)); !errors.Is(err, os.ErrProcessDone) {
				t.Fatalf("signal the rejected process: %v, want %v", err, os.ErrProcessDone)
			}
			if _, err := host.EnsureActive(t.Context(), "md.numen.engines.ocr"); !errors.Is(err, errPluginFailed) {
				t.Fatalf("second EnsureActive() = %v, want %v", err, errPluginFailed)
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
	host.SetLauncher(launchAnsweringPing(t, port.PluginProtocolVersion, &procs))

	if _, err := host.EnsureActive(t.Context(), "md.numen.engines.ocr"); err != nil {
		t.Fatalf("EnsureActive() = %v, want the plugin active", err)
	}
	host.Release("md.numen.engines.ocr")
	if err := host.TeardownAll(t.Context(), 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

func TestCheckPluginProtocolRefusesAProtocolTheHostDoesNotSpeak(t *testing.T) {
	newer := port.PluginProtocolVersion + 1
	endpoint := Endpoint{Host: "127.0.0.1", Port: servePing(t, int32(newer)), Token: ""}
	manifest := Manifest{ID: "md.numen.newer", Protocol: newer}

	err := checkPluginProtocol(t.Context(), manifest, endpoint)
	if !errors.Is(err, port.ErrUnsupportedProtocol) || !strings.Contains(err.Error(), "this host speaks 1") {
		t.Fatalf("checkPluginProtocol() = %v, want %v naming the host's protocol", err, port.ErrUnsupportedProtocol)
	}
}

func TestCheckPluginProtocolNamesAPluginThatDoesNotAnswerPing(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	addr, ok := srv.Listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("expected *net.TCPAddr, got %T", srv.Listener.Addr())
	}
	endpoint := Endpoint{Host: "127.0.0.1", Port: addr.Port, Token: ""}

	err := checkPluginProtocol(t.Context(), Manifest{ID: "md.numen.silent"}, endpoint)
	if err == nil || !strings.Contains(err.Error(), "ping plugin md.numen.silent") {
		t.Fatalf("checkPluginProtocol() = %v, want the ping failure naming the plugin", err)
	}
}
