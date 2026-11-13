package container_test

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

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultfixture"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
)

// meaningPluginID is a plugin that searches the notes by meaning.
const meaningPluginID = "md.numen.meaning"

// meaningPlugin describes itself as a source of eight-dimensional vectors and counts what it is asked.
type meaningPlugin struct {
	numenv1connect.UnimplementedSearchSourceServiceHandler
	recipe   atomic.Pointer[string]
	describe atomic.Int32
	texts    atomic.Int32
	searches atomic.Int32
}

func (p *meaningPlugin) DescribeSearchSource(
	context.Context, *connect.Request[v1.DescribeSearchSourceRequest],
) (*connect.Response[v1.DescribeSearchSourceResponse], error) {
	p.describe.Add(1)
	return connect.NewResponse(&v1.DescribeSearchSourceResponse{Name: "orchard-small", Recipe: *p.recipe.Load(), Width: 8, Concurrency: 1}), nil
}

func (p *meaningPlugin) IndexTexts(
	_ context.Context, req *connect.Request[v1.IndexTextsRequest],
) (*connect.Response[v1.IndexTextsResponse], error) {
	p.texts.Add(int32(len(req.Msg.GetTexts())))
	vectors := make([]*v1.FloatVector, len(req.Msg.GetTexts()))
	for i := range vectors {
		vectors[i] = &v1.FloatVector{Values: []float32{1, 0, 0, 0, 0, 0, 0, 0}}
	}
	return connect.NewResponse(&v1.IndexTextsResponse{Vectors: vectors}), nil
}

func (p *meaningPlugin) SearchPassages(
	context.Context, *connect.Request[v1.SearchSourceServiceSearchPassagesRequest],
) (*connect.Response[v1.SearchSourceServiceSearchPassagesResponse], error) {
	p.searches.Add(1)
	return connect.NewResponse(&v1.SearchSourceServiceSearchPassagesResponse{}), nil
}

// meaningCore is one process of the core over the meaning plugin: its plugin host and the times it started
// the plugin.
type meaningCore struct {
	roles    *container.PluginRoles
	config   container.Config
	launches *atomic.Int32
}

func openMeaningCore(t *testing.T, pluginDir, cacheRoot string, port int) meaningCore {
	t.Helper()
	roles := container.NewPluginRoles(testsupport.GetMachine(), "desktop.editor", pluginDir)
	if err := roles.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}
	roles.Storage.SetStorageRoots(cacheRoot, "")
	t.Cleanup(func() {
		if err := roles.Teardown.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond); err != nil {
			t.Error(err)
		}
	})
	// Each start takes the next of the processes started here.
	processes := []*exec.Cmd{wiretest.StartSleepProcess(t), wiretest.StartSleepProcess(t), wiretest.StartSleepProcess(t), wiretest.StartSleepProcess(t)}
	launches := &atomic.Int32{}
	roles.Processes.SetLauncher(func(_ context.Context, _ manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		next := int(launches.Add(1)) - 1
		if next >= len(processes) {
			return nil, nil, errors.New("no process is left to start")
		}
		return processes[next].Process, wiretest.GetReadyOutput(port), nil
	})
	return meaningCore{
		roles:    roles,
		config:   container.NewConfig(container.Config{PluginHost: container.AdaptPluginHost(roles), Paths: container.Paths{ServiceDir: ".numen"}}),
		launches: launches,
	}
}

func (c meaningCore) openSource(t *testing.T) searchsourceport.WatchedSource {
	t.Helper()
	source, _, closeSource, why := c.config.Embedders(t.Context())
	if why != nil {
		t.Fatal(why)
	}
	t.Cleanup(func() {
		if err := closeSource(); err != nil {
			t.Error(err)
		}
	})
	return source
}

func (c meaningCore) readVault(t *testing.T, db *container.Index, source searchsourceport.Source, v domain.Vault) {
	t.Helper()
	whole, err := container.ReadWholeVault(c.config, t.Context(), db, source, v)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := whole.Execute(t.Context(), v, indexing.Reporting{}); err != nil {
		t.Fatal(err)
	}
}

func (c meaningCore) awaitNoLease(t *testing.T, what string) {
	t.Helper()
	wiretest.WaitUntil(t, what, func() bool {
		sup, isKnown := c.roles.Activation.GetSupervisor(meaningPluginID)
		if !isKnown {
			return true
		}
		usage, _ := sup.GetUsage()
		return usage.LeaseCount == 0
	})
}

func serveMeaningPlugin(t *testing.T, plugin *meaningPlugin) (pluginDir string, port int) {
	t.Helper()
	plugin.setRecipe("orchard|small|8")
	mux := http.NewServeMux()
	wiretest.HandlePluginLifecycle(mux)
	mux.Handle(numenv1connect.NewSearchSourceServiceHandler(plugin))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	pluginDir = t.TempDir()
	writeMeaningManifest(t, pluginDir, "1.0.0")
	return pluginDir, wiretest.GetPort(t, server)
}

func writeMeaningManifest(t *testing.T, pluginDir, version string) {
	t.Helper()
	dir := filepath.Join(pluginDir, "meaning")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	text := `{"id": "` + meaningPluginID + `", "name": "Meaning", "version": "` + version + `",
		"capabilities": ["search.source"], "targets": ["desktop.editor"], "backend": {"entrypoint": "bin/meaning"}}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

// A core that once read the description of the search source opens a vault in the next process without
// starting the plugin, reads a vault with nothing to embed without starting it, starts it for new texts and
// for a search by meaning, gives it back after each, and never starts it for a search by words.
func TestTheSearchSourceRunsOnlyWhileThereIsWork(t *testing.T) {
	plugin := &meaningPlugin{}
	pluginDir, port := serveMeaningPlugin(t, plugin)
	cacheRoot := t.TempDir()
	db := makeIndex(t)
	v := vaultfixture.NewVault(t, map[string]string{"orchard.md": "# Orchard\n\napple trees bloom in the spring orchard\n"})

	first := openMeaningCore(t, pluginDir, cacheRoot, port)
	first.readVault(t, db, first.openSource(t), v)
	if err := first.roles.Teardown.StopPlugin(t.Context(), meaningPluginID); err != nil {
		t.Fatal(err)
	}

	next := openMeaningCore(t, pluginDir, cacheRoot, port)
	source := next.openSource(t)
	if got := next.launches.Load(); got != 0 {
		t.Fatalf("opening the vault started the plugin %d times, want none", got)
	}
	if source.Describe().SourceRecipe != "orchard|small|8" {
		t.Errorf("the source opened as %+v, want the description kept from the first process", source.Describe())
	}

	next.readVault(t, db, source, v)
	if got := next.launches.Load(); got != 0 {
		t.Fatalf("reading a vault with nothing to embed started the plugin %d times", got)
	}

	if _, err := container.NewSearch(next.config, db, source, nil).Execute(t.Context(), v, "orchard", search.GetTypingParameters(search.Lexical, 5)); err != nil {
		t.Fatal(err)
	}
	if got := next.launches.Load(); got != 0 {
		t.Fatalf("a search by words started the plugin %d times", got)
	}

	if err := os.WriteFile(filepath.Join(v.Path, "pears.md"), []byte("# Pears\n\npear trees ripen late in the autumn\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	texts := plugin.texts.Load()
	next.readVault(t, db, source, v)
	if next.launches.Load() != 1 || plugin.texts.Load() == texts {
		t.Fatalf("new texts started the plugin %d times and embedded %d texts, want one start", next.launches.Load(), plugin.texts.Load()-texts)
	}
	next.awaitNoLease(t, "the pass to give the plugin back")

	if err := next.roles.Teardown.StopPlugin(t.Context(), meaningPluginID); err != nil {
		t.Fatal(err)
	}
	if _, err := container.NewSearch(next.config, db, source, nil).Execute(t.Context(), v, "orchard", search.GetTypingParameters(search.Sources, 5)); err != nil {
		t.Fatal(err)
	}
	if next.launches.Load() != 2 || plugin.searches.Load() != 1 {
		t.Errorf("a search by meaning with the plugin stopped started it %d more times and asked it %d times, want once each", next.launches.Load()-1, plugin.searches.Load())
	}
	next.awaitNoLease(t, "the search to give the plugin back")
}

// A new version of the plugin asks it for its description again.
func TestANewVersionOfTheSearchSourceIsAskedForItsDescription(t *testing.T) {
	plugin := &meaningPlugin{}
	pluginDir, port := serveMeaningPlugin(t, plugin)
	cacheRoot := t.TempDir()
	first := openMeaningCore(t, pluginDir, cacheRoot, port)
	first.openSource(t)
	if err := first.roles.Teardown.StopPlugin(t.Context(), meaningPluginID); err != nil {
		t.Fatal(err)
	}

	writeMeaningManifest(t, pluginDir, "1.1.0")
	plugin.setRecipe("orchard|small|8|v1.1")
	next := openMeaningCore(t, pluginDir, cacheRoot, port)
	source := next.openSource(t)

	if got := next.launches.Load(); got != 1 {
		t.Errorf("the new version was started %d times, want once to describe itself", got)
	}
	if got := source.Describe().SourceRecipe; got != "orchard|small|8|v1.1" {
		t.Errorf("the source opened with recipe %q, want the one the new version gave", got)
	}
}

func (p *meaningPlugin) setRecipe(recipe string) { p.recipe.Store(&recipe) }
