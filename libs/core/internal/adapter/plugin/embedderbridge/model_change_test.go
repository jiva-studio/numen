package embedderbridge

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugintest"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

const (
	embeddingPluginID = "md.numen.embeddings"
	otherPluginID     = "md.numen.other"
)

// configurable is a plugin that takes its configuration.
type configurable struct {
	testsupport.PluginLifecycle
}

func (configurable) Configure(
	_ context.Context, _ *connect.Request[v1.ConfigureRequest],
) (*connect.Response[v1.ConfigureResponse], error) {
	return connect.NewResponse(&v1.ConfigureResponse{Success: true}), nil
}

// openConfigurable opens the bridge on a host that runs the embedding plugin and a second plugin.
func openConfigurable(t *testing.T) (*plugin.Host, *Embedder, *mockEmbeddingServer, *atomic.Int32) {
	t.Helper()
	mock := &mockEmbeddingServer{}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(configurable{}))
	mux.Handle(numenv1connect.NewEmbeddingServiceHandler(mock))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	port := plugintest.GetPort(t, server)

	root := t.TempDir()
	for id, capability := range map[string]string{embeddingPluginID: `"capabilities": ["embedding"],`, otherPluginID: ""} {
		dir := filepath.Join(root, strings.TrimPrefix(id, "md.numen."))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		text := `{"id": "` + id + `", "name": "` + id + `", "version": "1.0.0", ` + capability +
			`"targets": ["desktop.editor"], "backend": {"entrypoint": "bin/run"}}`
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	host := plugin.NewHost(testsupport.GetMachine(), "desktop.editor", root)
	if err := host.Discover(); err != nil {
		t.Fatal(err)
	}
	cmd := plugintest.StartSleepProcess(t)
	host.SetLauncher(func(_ context.Context, _ manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		return cmd.Process, plugintest.GetReadyOutput(port), nil
	})

	bridge, err := Open(t.Context(), host)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testsupport.Close(t, bridge) })

	var announced atomic.Int32
	bridge.OnModelChange(func() { announced.Add(1) })
	return host, bridge, mock, &announced
}

// settle waits until the read that follows a configuration has ended.
func settle(t *testing.T, bridge *Embedder, mock *mockEmbeddingServer, readsBefore int32) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for mock.getModelCalls.Load() == readsBefore {
		if time.Now().After(deadline) {
			t.Fatal("the model was not read after the plugin was configured")
		}
		time.Sleep(time.Millisecond)
	}
	bridge.changeMu.Lock()
	defer bridge.changeMu.Unlock()
}

// A model the plugin reports differently after a setting of its own is announced.
func TestAModelChangeAfterASettingOfThePluginIsAnnounced(t *testing.T) {
	host, bridge, mock, announced := openConfigurable(t)
	reads := mock.getModelCalls.Load()

	mock.modelName.Store("another-model")
	if err := host.UpdatePluginSetting(t.Context(), embeddingPluginID, "model", "another-model"); err != nil {
		t.Fatal(err)
	}
	settle(t, bridge, mock, reads)

	if got := announced.Load(); got != 1 {
		t.Errorf("the model change was announced %d times, want once", got)
	}
}

// A shorter minimum chunk length changes which chunks get a vector, so it is announced.
func TestAChangeOfTheMinimumChunkLengthIsAnnounced(t *testing.T) {
	host, bridge, mock, announced := openConfigurable(t)
	reads := mock.getModelCalls.Load()

	mock.minChunk.Store(7)
	if err := host.UpdatePluginSetting(t.Context(), embeddingPluginID, "min_chunk_length", 7); err != nil {
		t.Fatal(err)
	}
	settle(t, bridge, mock, reads)

	if got := announced.Load(); got != 1 {
		t.Errorf("the change was announced %d times, want once", got)
	}
}

// A setting of the plugin that leaves its model as it was announces nothing.
func TestASettingThatLeavesTheModelAsItWasAnnouncesNothing(t *testing.T) {
	host, bridge, mock, announced := openConfigurable(t)
	reads := mock.getModelCalls.Load()

	if err := host.UpdatePluginSetting(t.Context(), embeddingPluginID, "provider", "local"); err != nil {
		t.Fatal(err)
	}
	settle(t, bridge, mock, reads)

	if got := announced.Load(); got != 0 {
		t.Errorf("an unchanged model was announced %d times", got)
	}
}

// A setting of another plugin is not read by the bridge and announces nothing.
func TestASettingOfAnotherPluginAnnouncesNothing(t *testing.T) {
	host, bridge, mock, announced := openConfigurable(t)
	reads := mock.getModelCalls.Load()

	if err := host.UpdatePluginSetting(t.Context(), otherPluginID, "any", "value"); err != nil {
		t.Fatal(err)
	}
	// The plugin of the bridge is configured after the other one, with its model as it was. The
	// read this causes ends after any read the first write caused.
	if err := host.UpdatePluginSetting(t.Context(), embeddingPluginID, "provider", "local"); err != nil {
		t.Fatal(err)
	}
	settle(t, bridge, mock, reads)

	if got := mock.getModelCalls.Load() - reads; got != 1 {
		t.Errorf("the model was read %d times after two writes, want one for the plugin it belongs to", got)
	}
	if got := announced.Load(); got != 0 {
		t.Errorf("a write of another plugin announced a model change %d times", got)
	}
}

// A change read by GetModel before the plugin is configured again is still announced, because the
// listeners were told of the model they hold, not of the one the last read returned.
func TestAChangeReadByGetModelFirstIsStillAnnounced(t *testing.T) {
	host, bridge, mock, announced := openConfigurable(t)

	mock.modelName.Store("another-model")
	if got := bridge.GetModel().Name; got != "another-model" {
		t.Fatalf("GetModel reads %q", got)
	}
	reads := mock.getModelCalls.Load()
	if err := host.UpdatePluginSetting(t.Context(), embeddingPluginID, "model", "another-model"); err != nil {
		t.Fatal(err)
	}
	settle(t, bridge, mock, reads)

	if got := announced.Load(); got != 1 {
		t.Errorf("the model change was announced %d times, want once", got)
	}
}
