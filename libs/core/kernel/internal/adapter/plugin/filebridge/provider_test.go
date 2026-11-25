package filebridge_test

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
	"github.com/jiva-studio/numen/libs/shared/note/domain"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/filebridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
)

// storage is a plugin serving FileProviderService: it records the calls and answers them with err.
type storage struct {
	numenv1connect.UnimplementedFileProviderServiceHandler

	mu    sync.Mutex
	calls []string
	err   error
}

func (s *storage) FetchFile(
	_ context.Context, r *connect.Request[v1.FileProviderServiceFetchFileRequest],
) (*connect.Response[v1.FileProviderServiceFetchFileResponse], error) {
	s.record("fetch " + r.Msg.GetVaultId() + " " + r.Msg.GetPath())
	return connect.NewResponse(&v1.FileProviderServiceFetchFileResponse{}), s.err
}

func (s *storage) DropDownload(
	_ context.Context, r *connect.Request[v1.FileProviderServiceDropDownloadRequest],
) (*connect.Response[v1.FileProviderServiceDropDownloadResponse], error) {
	s.record("drop " + r.Msg.GetVaultId() + " " + r.Msg.GetPath())
	return connect.NewResponse(&v1.FileProviderServiceDropDownloadResponse{}), s.err
}

func (s *storage) GetFetchProgress(
	_ context.Context, r *connect.Request[v1.GetFetchProgressRequest],
) (*connect.Response[v1.GetFetchProgressResponse], error) {
	s.record("progress " + r.Msg.GetPath())
	return connect.NewResponse(&v1.GetFetchProgressResponse{IsKnown: true, BytesDone: 3 << 20, BytesTotal: 10 << 20}), s.err
}

func (s *storage) record(call string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, call)
}

func (s *storage) getCalls() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.calls...)
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
	host.Supervision.Processes.SetLauncher(func(_ context.Context, _ manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		return process.Process, wiretest.GetReadyOutput(port), nil
	})
	return host
}

const providerManifest = `{"id": "org.example.storage", "name": "Storage", "version": "1.0.0",
	"capabilities": ["files.provider"], "targets": ["desktop.editor"], "backend": {"entrypoint": "bin/run"}}`

func newProvider(host *pluginhost.Host) *filebridge.Provider {
	roles := pluginhost.Join(host)
	return filebridge.NewProvider(roles, roles)
}

func TestFetchingAFileCallsTheEnabledPluginThatProvidesFiles(t *testing.T) {
	plugin := &storage{}
	host := startHost(t, providerManifest, func(mux *http.ServeMux) {
		mux.Handle(numenv1connect.NewFileProviderServiceHandler(plugin))
	})

	err := newProvider(host).FetchFile(t.Context(), domain.Vault{ID: "v-1"}, "Books/Guide.pdf")

	if err != nil {
		t.Fatal(err)
	}
	if got := plugin.getCalls(); len(got) != 1 || got[0] != "fetch v-1 Books/Guide.pdf" {
		t.Errorf("the plugin was called with %q", got)
	}
}

func TestDropDownload_PluginFailure_ReturnsTheError(t *testing.T) {
	plugin := &storage{err: connect.NewError(connect.CodeFailedPrecondition, errors.New("The file has changes that are not sent"))}
	host := startHost(t, providerManifest, func(mux *http.ServeMux) {
		mux.Handle(numenv1connect.NewFileProviderServiceHandler(plugin))
	})

	err := newProvider(host).DropDownload(t.Context(), domain.Vault{ID: "v-1"}, "Books/Guide.pdf")

	var refusal *connect.Error
	if !errors.As(err, &refusal) || refusal.Code() != connect.CodeFailedPrecondition || refusal.Message() != "The file has changes that are not sent" {
		t.Errorf("the call failed with %v", err)
	}
	if got := plugin.getCalls(); len(got) != 1 || got[0] != "drop v-1 Books/Guide.pdf" {
		t.Errorf("the plugin was called with %q", got)
	}
}

func TestAFileCannotBeFetchedWhenNoPluginProvidesFiles(t *testing.T) {
	host := startHost(t, `{"id": "org.example.other", "name": "Other", "version": "1.0.0",
		"targets": ["desktop.editor"], "backend": {"entrypoint": "bin/run"}}`, func(*http.ServeMux) {})

	err := newProvider(host).FetchFile(t.Context(), domain.Vault{ID: "v-1"}, "a.pdf")

	if err == nil {
		t.Fatal("the fetch went through with no plugin to provide files")
	}
}

func TestGetFetchProgress_PluginAnswers_ReturnsTheBytesOfThePlugin(t *testing.T) {
	plugin := &storage{}
	host := startHost(t, providerManifest, func(mux *http.ServeMux) {
		mux.Handle(numenv1connect.NewFileProviderServiceHandler(plugin))
	})

	got, err := newProvider(host).GetFetchProgress(t.Context(), domain.Vault{ID: "v-1"}, "Books/Guide.pdf")

	if err != nil {
		t.Fatal(err)
	}
	if want := (file.DownloadProgress{IsKnown: true, BytesDone: 3 << 20, BytesTotal: 10 << 20}); got != want {
		t.Errorf("the progress is %+v, want %+v", got, want)
	}
	if calls := plugin.getCalls(); len(calls) != 1 || calls[0] != "progress Books/Guide.pdf" {
		t.Errorf("the plugin was called with %q", calls)
	}
}
