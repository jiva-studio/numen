package editor_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

const (
	gardenSourceID = "md.numen.embeddings"
	chooseRow      = "choose a connection in the settings of Vector Embeddings"
)

// connectionSource is the plugin edge of a search source served through a connection: it cannot describe
// itself while its settings name no connection row, and once they name one it indexes every text and
// answers every query with the chunks it indexed.
type connectionSource struct {
	wiretest.PluginLifecycle
	numenv1connect.UnimplementedSearchSourceServiceHandler

	mu         sync.Mutex
	connection string
	indexed    []string
}

func (s *connectionSource) Configure(
	_ context.Context, r *connect.Request[v1.ConfigureRequest],
) (*connect.Response[v1.ConfigureResponse], error) {
	s.mu.Lock()
	s.connection = r.Msg.GetValues()["connection"]
	s.mu.Unlock()
	return connect.NewResponse(&v1.ConfigureResponse{}), nil
}

func (s *connectionSource) DescribeSearchSource(
	context.Context, *connect.Request[v1.DescribeSearchSourceRequest],
) (*connect.Response[v1.DescribeSearchSourceResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.connection == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New(chooseRow))
	}
	return connect.NewResponse(&v1.DescribeSearchSourceResponse{
		Name: "garden-model", Recipe: "garden|" + s.connection + "|8", Width: 8, Concurrency: 1,
	}), nil
}

func (s *connectionSource) IndexTexts(
	_ context.Context, r *connect.Request[v1.IndexTextsRequest],
) (*connect.Response[v1.IndexTextsResponse], error) {
	vectors := make([]*v1.FloatVector, len(r.Msg.GetTexts()))
	s.mu.Lock()
	for i, text := range r.Msg.GetTexts() {
		s.indexed = append(s.indexed, text.GetChunkId())
		vectors[i] = &v1.FloatVector{Values: []float32{1, 0, 0, 0, 0, 0, 0, 0}}
	}
	s.mu.Unlock()
	return connect.NewResponse(&v1.IndexTextsResponse{Vectors: vectors}), nil
}

func (s *connectionSource) SearchChunks(
	context.Context, *connect.Request[v1.SearchSourceServiceSearchChunksRequest],
) (*connect.Response[v1.SearchSourceServiceSearchChunksResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	matches := make([]*v1.ChunkMatch, len(s.indexed))
	for i, chunk := range s.indexed {
		matches[i] = &v1.ChunkMatch{ChunkId: chunk, Score: 0.9}
	}
	return connect.NewResponse(&v1.SearchSourceServiceSearchChunksResponse{Matches: matches}), nil
}

func launchSleepProcesses(t *testing.T, port int) func(context.Context, manifest.Discovered, int, string) (*os.Process, io.ReadCloser, error) {
	t.Helper()
	var mu sync.Mutex
	var started []*os.Process
	var inputs []io.Closer
	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()
		for _, p := range started {
			wiretest.KillProcess(t, p)
			wiretest.ReapProcess(t, p)
		}
		for _, input := range inputs {
			if err := input.Close(); err != nil {
				t.Error(err)
			}
		}
	})
	return func(ctx context.Context, _ manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		cmd := exec.CommandContext(context.WithoutCancel(ctx), "/bin/sh", "-c", "read -r line")
		input, err := cmd.StdinPipe()
		if err != nil {
			return nil, nil, err
		}
		if err := cmd.Start(); err != nil {
			return nil, nil, err
		}
		mu.Lock()
		started = append(started, cmd.Process)
		inputs = append(inputs, input)
		mu.Unlock()
		return cmd.Process, wiretest.GetReadyOutput(port), nil
	}
}

func startConnectionSource(t *testing.T, plugin *connectionSource) *pluginhost.Host {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(plugin))
	mux.Handle(numenv1connect.NewSearchSourceServiceHandler(plugin))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	port := wiretest.GetPort(t, server)

	root := t.TempDir()
	dir := filepath.Join(root, "embeddings")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	text := `{"id": "` + gardenSourceID + `", "name": "Vector Embeddings", "version": "1.0.0", "capabilities": ["search.source"],
		"targets": ["desktop.editor"], "backend": {"entrypoint": "bin/embeddings"}}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	host := pluginhost.New(testsupport.GetMachine(), "desktop.editor", root)
	if err := host.Management.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}
	// The host forgets a plugin when its process exits and stops an idle plugin, so one process handed to two
	// starts ends the second start as soon as the stop of the first is noticed.
	host.Supervision.Processes.SetLauncher(launchSleepProcesses(t, port))
	return host
}

func (f *showing) findMeaning(t *testing.T, query string) []string {
	t.Helper()
	found, err := f.client.SearchText(t.Context(), connect.NewRequest(&v1.SearchTextRequest{
		Query: query, Mode: v1.SearchMode_SEARCH_MODE_MEANING,
	}))
	if err != nil {
		t.Fatal(err)
	}
	paths := make([]string, 0, len(found.Msg.GetResults()))
	for _, one := range found.Msg.GetResults() {
		paths = append(paths, one.GetPath())
	}
	return paths
}

func (f *showing) hasSearchItem(text string) bool {
	return slices.ContainsFunc(f.cfg.Execution.Health.List(), func(item plugindomain.HealthItem) bool {
		return item.PluginID == gardenSourceID && item.Key == plugindomain.HealthKeySearch && item.Severity == plugindomain.HealthError && item.Text == text
	})
}

// A window opened while the search source cannot describe itself shows why as the search item of its plugin;
// once a setting of the plugin names a connection, the source indexes the vault and answers search in that
// window, with no reopen, and the item goes.
func TestASourceThatDescribesItselfAfterASettingJoinsTheOpenWindow(t *testing.T) {
	plugin := &connectionSource{}
	host := startConnectionSource(t, plugin)
	cfg := newTestInstallation(container.Config{
		Paths: container.Paths{
			IndexPath:    filepath.Join(t.TempDir(), "index.db"),
			RegistryPath: filepath.Join(t.TempDir(), "vaults.json"),
		},
	})
	cfg.Execution, cfg.Preferences, cfg.Endpoints = container.CreatePluginHost(host.Management, host.Supervision)
	newListedVault(t, cfg, "garden", map[string]string{"Hedge.md": "# Hedge\n\nThe hedge along the lane is thick and tangled.\n"})

	installation, err := openRegisteredWindow(cfg, t.Context(), "garden", os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, installation)
	mux := http.NewServeMux()
	mountQuestionsServices(mux, installation)
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.CloseClientConnections)
	t.Cleanup(server.Close)
	f := &showing{client: newQuestionsClient(server.Client(), server.URL), installation: installation, cfg: cfg}
	f.read(t)

	if !f.hasSearchItem(chooseRow) {
		t.Errorf("the plugin has no search item %q: %+v", chooseRow, cfg.Execution.Health.List())
	}
	if found := f.findMeaning(t, "shrubbery"); len(found) != 0 {
		t.Errorf("a source that has not described itself found %v", found)
	}

	if err := cfg.Preferences.ApplySetting(t.Context(), gardenSourceID, "connection", "garden-row"); err != nil {
		t.Fatal(err)
	}

	testsupport.WaitFor(t, func() bool {
		return slices.Equal(slices.Compact(f.findMeaning(t, "shrubbery")), []string{"Hedge.md"})
	})
	if f.hasSearchItem(chooseRow) {
		t.Errorf("the plugin keeps the search item %q once the source described itself", chooseRow)
	}
}
