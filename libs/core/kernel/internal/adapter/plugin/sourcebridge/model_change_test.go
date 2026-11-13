package sourcebridge_test

import (
	"context"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/sourcebridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const (
	embeddingPluginID = "md.numen.embeddings"
	otherPluginID     = "md.numen.other"
)

// configurable is a plugin that takes its configuration.
type configurable struct {
	wiretest.PluginLifecycle
}

func (configurable) Configure(
	_ context.Context, _ *connect.Request[v1.ConfigureRequest],
) (*connect.Response[v1.ConfigureResponse], error) {
	return connect.NewResponse(&v1.ConfigureResponse{Success: true}), nil
}

func openConfigurableEmbedder(t *testing.T) (*container.PluginRoles, *sourcebridge.Source, *mockSourceServer, *atomic.Int32) {
	t.Helper()
	mock := &mockSourceServer{}
	mock.chunkTokens.Store(256)
	host := startConfigurableSource(t, mock)

	roles := pluginhost.Join(host)
	bridge, err := sourcebridge.Open(t.Context(), roles, roles, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testsupport.Close(t, bridge) })

	var announced atomic.Int32
	bridge.OnDescriptionChange(func() { announced.Add(1) })
	return host, bridge, mock, &announced
}

func startConfigurableSource(t *testing.T, mock *mockSourceServer) *container.PluginRoles {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(configurable{}))
	mux.Handle(numenv1connect.NewSearchSourceServiceHandler(mock))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	port := wiretest.GetPort(t, server)

	root := t.TempDir()
	for id, capability := range map[string]string{embeddingPluginID: `"capabilities": ["search.source"],`, otherPluginID: ""} {
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
	host := container.NewPluginRoles(testsupport.GetMachine(), "desktop.editor", root)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}
	cmd := wiretest.StartSleepProcess(t)
	host.Processes.SetLauncher(func(_ context.Context, _ manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		return cmd.Process, wiretest.GetReadyOutput(port), nil
	})
	return host
}

func settle(t *testing.T, bridge *sourcebridge.Source, mock *mockSourceServer, readsBefore int32) {
	t.Helper()
	wiretest.WaitUntil(t, "a read of the model after the plugin was configured", func() bool {
		return mock.describeCalls.Load() != readsBefore
	})
	bridge.WaitForAnnouncements()
}

// A model the plugin reports differently after a setting of its own is announced.
func TestAModelChangeAfterASettingOfThePluginIsAnnounced(t *testing.T) {
	host, bridge, mock, announced := openConfigurableEmbedder(t)
	reads := mock.describeCalls.Load()

	mock.modelName.Store("another-model")
	if err := updateSetting(t.Context(), host, embeddingPluginID, "model", "another-model"); err != nil {
		t.Fatal(err)
	}
	settle(t, bridge, mock, reads)

	if got := announced.Load(); got != 1 {
		t.Errorf("the model change was announced %d times, want once", got)
	}
}

// A shorter minimum chunk length changes which chunks get a vector, so it is announced.
func TestAChangeOfTheMinimumChunkLengthIsAnnounced(t *testing.T) {
	host, bridge, mock, announced := openConfigurableEmbedder(t)
	reads := mock.describeCalls.Load()

	mock.minText.Store(7)
	if err := updateSetting(t.Context(), host, embeddingPluginID, "minChunkLength", 7); err != nil {
		t.Fatal(err)
	}
	settle(t, bridge, mock, reads)

	if got := announced.Load(); got != 1 {
		t.Errorf("the change was announced %d times, want once", got)
	}
}

// A setting of the plugin that leaves its model as it was announces nothing.
func TestNoModelChangeIsAnnouncedWhenASettingKeepsTheModel(t *testing.T) {
	host, bridge, mock, announced := openConfigurableEmbedder(t)
	reads := mock.describeCalls.Load()

	if err := updateSetting(t.Context(), host, embeddingPluginID, "provider", "local"); err != nil {
		t.Fatal(err)
	}
	settle(t, bridge, mock, reads)

	if got := announced.Load(); got != 0 {
		t.Errorf("an unchanged model was announced %d times", got)
	}
}

// A setting of another plugin is not read by the bridge and announces nothing.
func TestASettingOfAnotherPluginAnnouncesNothing(t *testing.T) {
	host, bridge, mock, announced := openConfigurableEmbedder(t)
	reads := mock.describeCalls.Load()

	if err := updateSetting(t.Context(), host, otherPluginID, "any", "value"); err != nil {
		t.Fatal(err)
	}
	// The plugin of the bridge is configured after the other one, with its model as it was. The
	// read this causes ends after any read the first write caused.
	if err := updateSetting(t.Context(), host, embeddingPluginID, "provider", "local"); err != nil {
		t.Fatal(err)
	}
	settle(t, bridge, mock, reads)

	if got := mock.describeCalls.Load() - reads; got != 1 {
		t.Errorf("the model was read %d times after two writes, want one for the plugin it belongs to", got)
	}
	if got := announced.Load(); got != 0 {
		t.Errorf("a write of another plugin announced a model change %d times", got)
	}
}

// The chunk size the plugin reports is read with the model.
func TestTheModelCarriesTheChunkTokensThePluginReports(t *testing.T) {
	_, bridge, _, _ := openConfigurableEmbedder(t)

	if got := bridge.Describe().ChunkTokens; got != 256 {
		t.Errorf("ChunkTokens = %d, want 256", got)
	}
}

// A chunk size below zero is a plugin that names none.
func TestANegativeChunkTokensIsReadAsNone(t *testing.T) {
	host, bridge, mock, _ := openConfigurableEmbedder(t)
	reads := mock.describeCalls.Load()

	mock.chunkTokens.Store(-5)
	if err := updateSetting(t.Context(), host, embeddingPluginID, "chunkTokens", -5); err != nil {
		t.Fatal(err)
	}
	settle(t, bridge, mock, reads)

	if got := bridge.Describe().ChunkTokens; got != 0 {
		t.Errorf("ChunkTokens = %d, want 0", got)
	}
}

// A new chunk size cuts the notes again, so a change of it alone is announced, once, and the model
// the listeners read holds the new size.
func TestAChangeOfTheChunkTokensAloneIsAnnounced(t *testing.T) {
	host, bridge, mock, announced := openConfigurableEmbedder(t)
	reads := mock.describeCalls.Load()

	mock.chunkTokens.Store(128)
	if err := updateSetting(t.Context(), host, embeddingPluginID, "chunkTokens", 128); err != nil {
		t.Fatal(err)
	}
	settle(t, bridge, mock, reads)

	if got := announced.Load(); got != 1 {
		t.Errorf("the change of the chunk size was announced %d times, want once", got)
	}
	if got := bridge.Describe().ChunkTokens; got != 128 {
		t.Errorf("ChunkTokens = %d, want 128", got)
	}
}

// A chunk size that stays as it was announces nothing.
func TestAnUnchangedChunkTokensAnnouncesNothing(t *testing.T) {
	host, bridge, mock, announced := openConfigurableEmbedder(t)
	reads := mock.describeCalls.Load()

	if err := updateSetting(t.Context(), host, embeddingPluginID, "chunkTokens", 256); err != nil {
		t.Fatal(err)
	}
	settle(t, bridge, mock, reads)

	if got := announced.Load(); got != 0 {
		t.Errorf("an unchanged chunk size was announced %d times", got)
	}
}
