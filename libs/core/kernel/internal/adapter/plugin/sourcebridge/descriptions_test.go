package sourcebridge_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/sourcebridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"
)

// keptSource is a search source plugin whose host keeps descriptions in a cache folder, and counts the times
// it started the plugin.
type keptSource struct {
	host     *container.PluginRoles
	mock     *mockSourceServer
	launches *atomic.Int32
}

func startKeptSource(t *testing.T) keptSource {
	t.Helper()
	mock := &mockSourceServer{}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(configurable{}))
	mux.Handle(numenv1connect.NewSearchSourceServiceHandler(mock))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	port := wiretest.GetPort(t, server)

	root := t.TempDir()
	dir := filepath.Join(root, "embeddings")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	text := `{"id": "` + embeddingPluginID + `", "name": "Vector Embeddings", "version": "1.0.0",
		"capabilities": ["search.source"], "targets": ["desktop.editor"], "backend": {"entrypoint": "bin/run"},
		"settings": [{"key": "model", "type": "string", "title": "Model", "default": "small"}]}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	host := container.NewPluginRoles(testsupport.GetMachine(), "desktop.editor", root)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}
	host.Storage.SetStorageRoots(t.TempDir(), "")
	t.Cleanup(func() {
		if err := host.Teardown.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond); err != nil {
			t.Error(err)
		}
	})
	// Each start takes the next of the processes started here.
	processes := []*exec.Cmd{wiretest.StartSleepProcess(t), wiretest.StartSleepProcess(t), wiretest.StartSleepProcess(t), wiretest.StartSleepProcess(t)}
	launches := &atomic.Int32{}
	host.Processes.SetLauncher(func(_ context.Context, _ manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		next := int(launches.Add(1)) - 1
		if next >= len(processes) {
			return nil, nil, errors.New("no process is left to start")
		}
		return processes[next].Process, wiretest.GetReadyOutput(port), nil
	})
	return keptSource{host: host, mock: mock, launches: launches}
}

func (s keptSource) open(t *testing.T, kept *sourcebridge.Descriptions) *sourcebridge.Source {
	t.Helper()
	roles := pluginhost.Join(s.host)
	bridge, err := sourcebridge.Open(t.Context(), roles, roles, kept)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testsupport.Close(t, bridge) })
	return bridge
}

func (s keptSource) stop(t *testing.T) {
	t.Helper()
	if err := s.host.Teardown.StopPlugin(t.Context(), embeddingPluginID); err != nil {
		t.Fatal(err)
	}
}

// A source opened again under the configuration it was read under takes the description it gave, from
// memory and, in a new process, from its file, and starts no plugin.
func TestReopeningASourceWithTheSameConfigurationStartsNoPlugin(t *testing.T) {
	source := startKeptSource(t)
	source.mock.recipe.Store("orchard|small|384")
	first := source.open(t, sourcebridge.NewDescriptions(source.host.Storage, source.host.Config)).Describe()
	source.stop(t)
	reads, launches := source.mock.describeCalls.Load(), source.launches.Load()

	again := source.open(t, sourcebridge.NewDescriptions(source.host.Storage, source.host.Config)).Describe()

	if got := source.launches.Load() - launches; got != 0 {
		t.Errorf("opening the source again started the plugin %d times", got)
	}
	if got := source.mock.describeCalls.Load() - reads; got != 0 {
		t.Errorf("opening the source again asked for its description %d times", got)
	}
	if again != first || again.SourceRecipe != "orchard|small|384" {
		t.Errorf("the source opened again as %+v, want %+v", again, first)
	}
}

// A setting that changes while the plugin is stopped starts it to read the description again, announces the
// new model, and keeps it for the next opening.
func TestASettingChangedWhileThePluginIsStoppedIsReadAgain(t *testing.T) {
	source := startKeptSource(t)
	kept := sourcebridge.NewDescriptions(source.host.Storage, source.host.Config)
	bridge := source.open(t, kept)
	var announced atomic.Int32
	bridge.OnDescriptionChange(func() { announced.Add(1) })
	source.stop(t)

	source.mock.modelName.Store("large")
	reads := source.mock.describeCalls.Load()
	if err := updateSetting(t.Context(), source.host, embeddingPluginID, "model", "large"); err != nil {
		t.Fatal(err)
	}
	wiretest.WaitUntil(t, "the new model to be read and announced", func() bool {
		return source.mock.describeCalls.Load()-reads >= 1 && announced.Load() == 1
	})
	bridge.WaitForAnnouncements()
	if got := source.mock.describeCalls.Load() - reads; got != 1 {
		t.Errorf("the change was read %d times, want once: the start the read caused reads nothing again", got)
	}
	source.stop(t)
	launches := source.launches.Load()

	reopened := source.open(t, sourcebridge.NewDescriptions(source.host.Storage, source.host.Config))
	if got := reopened.Describe().Name; got != "large" {
		t.Errorf("the source opened after the change as %q, want the new model", got)
	}
	if got := source.launches.Load() - launches; got != 0 {
		t.Errorf("opening the source after the change started the plugin %d times", got)
	}
}
