package editor_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const (
	gardenSourceID = "md.numen.embeddings"
	chooseRow      = "choose a connection in the settings of Vector Embeddings"
	gardenIndexed  = "every chunk of the vault is indexed"
	gardenPending  = "the index of the vault is being built"
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
	var settings struct {
		Connection string `json:"connection"`
	}
	if err := json.Unmarshal([]byte(r.Msg.GetConfigurationJson()), &settings); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	s.mu.Lock()
	s.connection = settings.Connection
	s.mu.Unlock()
	return connect.NewResponse(&v1.ConfigureResponse{Success: true}), nil
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

func (s *connectionSource) SearchPassages(
	context.Context, *connect.Request[v1.SearchSourceServiceSearchPassagesRequest],
) (*connect.Response[v1.SearchSourceServiceSearchPassagesResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	matches := make([]*v1.ChunkMatch, len(s.indexed))
	for i, chunk := range s.indexed {
		matches[i] = &v1.ChunkMatch{ChunkId: chunk, Score: 0.9}
	}
	return connect.NewResponse(&v1.SearchSourceServiceSearchPassagesResponse{Matches: matches}), nil
}

func (s *connectionSource) DescribeCoverage(
	_ context.Context, r *connect.Request[v1.DescribeCoverageRequest],
) (*connect.Response[v1.DescribeCoverageResponse], error) {
	text := gardenPending
	if r.Msg.GetChunkCount() > 0 && r.Msg.GetIndexedCount() == r.Msg.GetChunkCount() {
		text = gardenIndexed
	}
	return connect.NewResponse(&v1.DescribeCoverageResponse{Text: text}), nil
}

func startConnectionSource(t *testing.T, plugin *connectionSource) *container.PluginRoles {
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
	host := container.NewPluginRoles(testsupport.GetMachine(), "desktop.editor", root)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}
	cmd := wiretest.StartSleepProcess(t)
	host.Processes.SetLauncher(func(context.Context, manifest.Discovered, int, string) (*os.Process, io.ReadCloser, error) {
		return cmd.Process, wiretest.GetReadyOutput(port), nil
	})
	return host
}

func (f *showing) findMeaning(t *testing.T, query string) []string {
	t.Helper()
	found, err := f.client.SearchPassages(t.Context(), connect.NewRequest(&v1.SearchPassagesRequest{
		Query: query, Mode: v1.SearchMode_SEARCH_MODE_MEANING,
	}))
	if err != nil {
		t.Fatal(err)
	}
	paths := make([]string, 0, len(found.Msg.GetFound()))
	for _, one := range found.Msg.GetFound() {
		paths = append(paths, one.GetPath())
	}
	return paths
}

func (f *showing) hasTaskError(text string) bool {
	return slices.ContainsFunc(f.installation.State.Window.Tasks.List(), func(task work.Task) bool { return task.Error == text })
}

// A window opened while the search source cannot describe itself shows why; once a setting of its plugin
// names a connection, the source indexes the vault and answers search in that window, with no reopen.
func TestASourceThatDescribesItselfAfterASettingJoinsTheOpenWindow(t *testing.T) {
	plugin := &connectionSource{}
	host := startConnectionSource(t, plugin)
	cfg := container.NewConfig(container.Config{
		Paths: container.Paths{IndexPath: filepath.Join(t.TempDir(), "index.db"), RegistryPath: filepath.Join(t.TempDir(), "vaults.json")},
	})
	cfg.PluginHost = container.AdaptPluginHost(host)
	newListedVault(t, cfg, "garden", map[string]string{"Hedge.md": "# Hedge\n\nThe hedge along the lane is thick and tangled.\n"})

	installation, _, err := openRegisteredWindow(cfg, t.Context(), "garden", os.Stderr)
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

	if !f.hasTaskError(chooseRow) {
		t.Errorf("the window does not list %q among its tasks: %+v", chooseRow, installation.State.Window.Tasks.List())
	}
	if found := f.findMeaning(t, "shrubbery"); len(found) != 0 {
		t.Errorf("a source that has not described itself found %v", found)
	}
	if coverage := f.state(t).GetCoverage().GetText(); coverage != "" {
		t.Errorf("a source that has not described itself has the coverage %q", coverage)
	}

	if err := cfg.PluginHost.ApplySetting(t.Context(), gardenSourceID, "connection", "garden-row"); err != nil {
		t.Fatal(err)
	}

	testsupport.WaitFor(t, func() bool { return f.state(t).GetCoverage().GetText() == gardenIndexed })
	if found := f.findMeaning(t, "shrubbery"); !slices.Equal(slices.Compact(found), []string{"Hedge.md"}) {
		t.Errorf("the source found %v, want Hedge.md", found)
	}
	if f.hasTaskError(chooseRow) {
		t.Errorf("the window still lists %q once the source described itself", chooseRow)
	}
}
