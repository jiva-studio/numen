package vaultbridge_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/vaultbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"
)

// cloud is a plugin serving VaultProviderService: it offers two remote vaults, records the connections and
// answers them with err.
type cloud struct {
	numenv1connect.UnimplementedVaultProviderServiceHandler

	mu    sync.Mutex
	calls []string
	err   error
}

func (c *cloud) ListRemoteVaults(
	context.Context, *connect.Request[v1.ListRemoteVaultsRequest],
) (*connect.Response[v1.ListRemoteVaultsResponse], error) {
	return connect.NewResponse(&v1.ListRemoteVaultsResponse{Vaults: []*v1.RemoteVault{
		{RemoteId: "r-1", Name: "Notes", SizeBytes: 4096, VaultId: "v-1"},
		{RemoteId: "r-2", Name: "Books"},
	}}), c.err
}

func (c *cloud) ConnectRemoteVault(
	_ context.Context, r *connect.Request[v1.ConnectRemoteVaultRequest],
) (*connect.Response[v1.ConnectRemoteVaultResponse], error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls = append(c.calls, "connect "+r.Msg.GetRemoteId()+" "+r.Msg.GetVaultId())
	return connect.NewResponse(&v1.ConnectRemoteVaultResponse{}), c.err
}

func (c *cloud) getCalls() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.calls...)
}

func startHost(t *testing.T, manifestText string, handle func(*http.ServeMux)) *pluginhost.Host {
	t.Helper()
	mux := http.NewServeMux()
	wiretest.HandlePluginLifecycle(mux)
	handle(mux)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	port := wiretest.GetPort(t, server)

	root := t.TempDir()
	dir := filepath.Join(root, "sync")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifestText), 0o600); err != nil {
		t.Fatal(err)
	}
	host := pluginhost.New(testsupport.GetMachine(), "desktop.editor", root)
	if err := host.Management.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}
	host.Supervision.Storage.SetStorageRoots(t.TempDir(), "")
	t.Cleanup(func() {
		if err := host.Management.Teardown.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond); err != nil {
			t.Error(err)
		}
	})
	process := wiretest.StartSleepProcess(t)
	host.Supervision.Processes.SetLauncher(func(_ context.Context, _ manifest.DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		return process.Process, wiretest.GetReadyOutput(port), nil
	})
	return host
}

const providerManifest = `{"id": "org.example.cloud", "name": "Cloud", "version": "1.0.0",
	"capabilities": ["vault.provider"], "targets": ["desktop.editor"], "backend": {"entrypoint": "bin/run"}}`

func newProvider(host *pluginhost.Host) *vaultbridge.Provider {
	roles := pluginhost.Join(host)
	return vaultbridge.NewProvider(roles, roles)
}

func TestListingRemoteVaultsCallsTheEnabledPluginThatProvidesVaults(t *testing.T) {
	plugin := &cloud{}
	host := startHost(t, providerManifest, func(mux *http.ServeMux) {
		mux.Handle(numenv1connect.NewVaultProviderServiceHandler(plugin))
	})

	got, err := newProvider(host).ListRemoteVaults(t.Context())

	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].RemoteID != "r-1" || got[0].Name != "Notes" || got[0].SizeBytes != 4096 ||
		got[0].VaultID != "v-1" || got[1].RemoteID != "r-2" || got[1].VaultID != "" {
		t.Errorf("the plugin listed %+v", got)
	}
}

func TestConnectRemoteVault_PluginFailure_ReturnsTheError(t *testing.T) {
	plugin := &cloud{err: connect.NewError(connect.CodeFailedPrecondition, errors.New("Sign in to connect the vault"))}
	host := startHost(t, providerManifest, func(mux *http.ServeMux) {
		mux.Handle(numenv1connect.NewVaultProviderServiceHandler(plugin))
	})

	err := newProvider(host).ConnectRemoteVault(t.Context(), "r-1", "v-9")

	var refusal *connect.Error
	if !errors.As(err, &refusal) || refusal.Code() != connect.CodeFailedPrecondition || refusal.Message() != "Sign in to connect the vault" {
		t.Errorf("the call failed with %v", err)
	}
	if got := plugin.getCalls(); len(got) != 1 || got[0] != "connect r-1 v-9" {
		t.Errorf("the plugin was called with %q", got)
	}
}

func TestRemoteVaultsCannotBeListedWhenNoPluginProvidesThem(t *testing.T) {
	host := startHost(t, `{"id": "org.example.other", "name": "Other", "version": "1.0.0",
		"targets": ["desktop.editor"], "backend": {"entrypoint": "bin/run"}}`, func(*http.ServeMux) {})

	_, err := newProvider(host).ListRemoteVaults(t.Context())

	if err == nil {
		t.Fatal("the list went through with no plugin to provide vaults")
	}
}
