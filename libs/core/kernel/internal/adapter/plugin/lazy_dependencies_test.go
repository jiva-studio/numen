package plugin_test

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/storage"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/toolbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/provision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	catalogmanifest "github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const (
	lazyCaller      = "md.example.caller"
	lazyToolName    = "note_read"
	lazyDownloadMax = 5 * time.Second
)

type modelSource struct {
	dir         string
	mu          sync.Mutex
	downloads   int
	failure     error
	gate        chan struct{}
	downloading chan struct{}
	duringFetch func()
}

func newModelSource(t *testing.T) *modelSource {
	t.Helper()
	return &modelSource{dir: t.TempDir(), downloading: make(chan struct{}, 32)}
}

func (s *modelSource) envName(id string) string {
	return "NUMEN_DEPENDENCY_" + strings.ToUpper(strings.ReplaceAll(id, "-", "_"))
}

func (s *modelSource) path(id string) string { return filepath.Join(s.dir, id+".bin") }

func (s *modelSource) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.downloads
}

func (s *modelSource) setFailure(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failure = err
}

func (s *modelSource) holdDownloads(t *testing.T) (release func()) {
	t.Helper()
	s.mu.Lock()
	s.gate = make(chan struct{})
	gate := s.gate
	s.mu.Unlock()
	var once sync.Once
	release = func() { once.Do(func() { close(gate) }) }
	t.Cleanup(release)
	return release
}

func (s *modelSource) waitForDownload(t *testing.T) {
	t.Helper()
	select {
	case <-s.downloading:
	case <-time.After(lazyDownloadMax):
		t.Fatal("no download started")
	}
}

func (s *modelSource) CanProvision(catalogmanifest.DependencyRequirement) bool { return true }

func (s *modelSource) Resolve(req catalogmanifest.DependencyRequirement) (plugindomain.ResolvedDependency, plugindomain.DependencyState, error) {
	path := s.path(req.ID)
	resolved := plugindomain.ResolvedDependency{
		ID: req.ID, RootPath: s.dir, PrimaryPath: path, EnvVars: map[string]string{s.envName(req.ID): path},
	}
	state := plugindomain.DependencyStateReady
	if _, err := os.Stat(path); err != nil {
		state = plugindomain.DependencyStateMissing
	}
	return resolved, state, nil
}

func (s *modelSource) Provision(
	ctx context.Context, req catalogmanifest.DependencyRequirement, onProgress func(plugindomain.DependencyStatus),
) (plugindomain.ResolvedDependency, error) {
	s.mu.Lock()
	s.downloads++
	failure, gate, duringFetch := s.failure, s.gate, s.duringFetch
	s.mu.Unlock()
	select {
	case s.downloading <- struct{}{}:
	default:
	}
	if duringFetch != nil {
		duringFetch()
	}
	onProgress(plugindomain.DependencyStatus{BytesDownloaded: 1, BytesTotal: 2, Message: "Downloading " + req.ID})
	if gate != nil {
		select {
		case <-gate:
		case <-ctx.Done():
			return plugindomain.ResolvedDependency{}, ctx.Err()
		}
	}
	if failure != nil {
		return plugindomain.ResolvedDependency{}, failure
	}
	if err := os.WriteFile(s.path(req.ID), []byte("model "+req.ID), 0o644); err != nil {
		return plugindomain.ResolvedDependency{}, err
	}
	resolved, _, err := s.Resolve(req)
	return resolved, err
}

type readOnlyTools struct {
	numenv1connect.UnimplementedPluginToolServiceHandler
}

func (readOnlyTools) ListTools(
	context.Context, *connect.Request[v1.ListToolsRequest],
) (*connect.Response[v1.ListToolsResponse], error) {
	return connect.NewResponse(&v1.ListToolsResponse{Tools: []*v1.PluginTool{{Name: lazyToolName, Effect: v1.ToolEffect_TOOL_EFFECT_READ_ONLY}}}), nil
}

func (readOnlyTools) CallTool(
	_ context.Context, req *connect.Request[v1.CallToolRequest],
) (*connect.Response[v1.CallToolResponse], error) {
	return connect.NewResponse(&v1.CallToolResponse{Result: &v1.ToolResult{Text: req.Msg.GetName()}}), nil
}

func serveReadOnlyTools(backend *builtInBackend) func(map[string]string) (http.Handler, error) {
	return func(vars map[string]string) (http.Handler, error) {
		backend.mu.Lock()
		backend.vars = append(backend.vars, vars)
		backend.mu.Unlock()
		mux := http.NewServeMux()
		mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(backend.probe))
		mux.Handle(numenv1connect.NewPluginToolServiceHandler(readOnlyTools{}))
		return mux, nil
	}
}

type toolsInfoHost struct{ pluginID string }

func (h toolsInfoHost) GetPluginInfos() []plugindomain.Info {
	return []plugindomain.Info{
		{ID: h.pluginID, Version: "1.0.0", Capabilities: []string{"tools"}, Permissions: []string{"vault:read"}},
		{ID: lazyCaller, Version: "1.0.0", Permissions: []string{"vault:read"}},
	}
}

func (h toolsInfoHost) IsEnabled(string) bool { return true }

type lazyPlugin struct {
	host    *pluginhost.Host
	backend *builtInBackend
	catalog *toolbridge.Catalog
}

func newLazyPluginFrom(t *testing.T, dependencies *provision.Registry, manifest []byte) lazyPlugin {
	t.Helper()
	host := newBuiltInHost(t)
	host.Supervision.Dependencies.Replace(dependencies)
	return newLazyPluginOn(t, host, manifest)
}

func newLazyPluginOn(t *testing.T, host *pluginhost.Host, manifest []byte) lazyPlugin {
	t.Helper()
	backend := newBuiltInBackend()
	registerBuiltIn(t, host.Management, manifest, serveReadOnlyTools(backend))
	files := storage.New(func(err error) { t.Errorf("the storage reported %v", err) }, func(string) string { return "1.0.0" })
	files.SetStorageRoots(t.TempDir(), "")
	catalog := toolbridge.NewCatalog(host.Management.Activation, toolsInfoHost{pluginID: firstBuiltInID}, files, func() string { return "r1" }, testsupport.NewClock(), io.Discard)
	return lazyPlugin{host: host, backend: backend, catalog: catalog}
}

func newLazyPlugin(t *testing.T, source *modelSource) lazyPlugin {
	t.Helper()
	t.Setenv(source.envName(sampleRuntime), "")
	host := newBuiltInHost(t)
	host.Supervision.Dependencies.Replace(provision.NewObservedRegistry(host.Supervision.Services.Observe, source))
	return newLazyPluginOn(t, host, newBuiltInManifest(firstBuiltInID, sampleRuntime))
}

func (p lazyPlugin) listTools(t *testing.T) []string {
	t.Helper()
	var names []string
	for _, one := range p.catalog.ListTools(t.Context(), lazyCaller) {
		names = append(names, one.GetName())
	}
	return names
}

func (p lazyPlugin) use(t *testing.T) error {
	t.Helper()
	endpoint, err := p.host.Management.Activation.EnsureActive(t.Context(), firstBuiltInID, getTestHolder())
	if err != nil {
		return err
	}
	p.host.Management.Activation.Release(firstBuiltInID, endpoint.Token, getTestHolder())
	return nil
}

func (p lazyPlugin) waitUntilReleased(t *testing.T) {
	t.Helper()
	wiretest.WaitUntil(t, "the plugin to hold no lease", func() bool {
		return slices.Contains(p.host.Management.Teardown.EvictIdle(t.Context(), 0, releaseGrace), firstBuiltInID)
	})
}

func TestAPluginStartedForAQuestionDownloadsNothing(t *testing.T) {
	source := newModelSource(t)
	p := newLazyPlugin(t, source)

	endpoint, err := p.host.Management.Activation.EnsureActiveForQuestion(t.Context(), firstBuiltInID, getTestHolder())
	if err != nil {
		t.Fatal(err)
	}
	p.host.Management.Activation.ReleaseQuestion(firstBuiltInID, endpoint.Token, getTestHolder())

	if got := source.count(); got != 0 {
		t.Errorf("the question downloaded %d times, want none", got)
	}
}

func TestListingToolsStartsAPluginWithoutDownloadingItsModels(t *testing.T) {
	source := newModelSource(t)
	p := newLazyPlugin(t, source)

	names := p.listTools(t)

	if !slices.Equal(names, []string{lazyToolName}) {
		t.Fatalf("the listing gave %v", names)
	}
	if got := source.count(); got != 0 {
		t.Errorf("the listing downloaded %d times, want none", got)
	}
	if _, err := os.Stat(source.path(sampleRuntime)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the listing left a model file: %v", err)
	}
}

func TestListingToolsDoesNotWaitForADownloadInProgress(t *testing.T) {
	source := newModelSource(t)
	release := source.holdDownloads(t)
	p := newLazyPlugin(t, source)
	using := make(chan error, 1)
	go func() { using <- p.use(t) }()
	source.waitForDownload(t)
	listed := make(chan []string, 1)

	go func() { listed <- p.listTools(t) }()

	select {
	case names := <-listed:
		if !slices.Equal(names, []string{lazyToolName}) {
			t.Errorf("the listing gave %v", names)
		}
	case <-time.After(3 * time.Second):
		t.Error("the listing waited for the download")
	}
	release()
	if err := <-using; err != nil {
		t.Errorf("the use failed: %v", err)
	}
}

func TestTheFirstUseDownloadsTheModelsOfThePlugin(t *testing.T) {
	source := newModelSource(t)
	p := newLazyPlugin(t, source)

	if err := p.use(t); err != nil {
		t.Fatal(err)
	}

	if got := source.count(); got != 1 {
		t.Fatalf("the first use downloaded %d times, want 1", got)
	}
	if _, err := os.Stat(source.path(sampleRuntime)); err != nil {
		t.Errorf("the model is not at its path after the use: %v", err)
	}
}

func TestAPluginIsStartedWithThePlannedPathOfAModelThatIsNotThereYet(t *testing.T) {
	source := newModelSource(t)
	p := newLazyPlugin(t, source)

	if err := p.use(t); err != nil {
		t.Fatal(err)
	}

	if got := p.backend.lastVars(t)[source.envName(sampleRuntime)]; got != source.path(sampleRuntime) {
		t.Errorf("the plugin started with %s = %q, want the planned path %q", source.envName(sampleRuntime), got, source.path(sampleRuntime))
	}
}

func TestTheDownloadStartsAfterThePluginIsConfigured(t *testing.T) {
	source := newModelSource(t)
	p := newLazyPlugin(t, source)
	var configuredAtDownload atomic.Int32
	source.duringFetch = func() { configuredAtDownload.Store(int32(p.backend.probe.count())) }

	if err := p.use(t); err != nil {
		t.Fatal(err)
	}

	if configuredAtDownload.Load() == 0 {
		t.Error("the download began before the plugin took its configuration")
	}
}

func TestAToolCallAfterTheListingDownloadsTheModels(t *testing.T) {
	source := newModelSource(t)
	p := newLazyPlugin(t, source)
	p.listTools(t)
	if got := source.count(); got != 0 {
		t.Fatalf("the listing downloaded %d times", got)
	}

	res, err := p.catalog.CallTool(t.Context(), lazyCaller, &v1.CallToolRequest{Name: lazyToolName})

	if err != nil {
		t.Fatal(err)
	}
	if res.GetResult().GetText() != lazyToolName {
		t.Errorf("the call answered %v", res)
	}
	if got := source.count(); got != 1 {
		t.Errorf("the call downloaded %d times, want 1", got)
	}
}

func TestASecondUseDownloadsNothing(t *testing.T) {
	source := newModelSource(t)
	p := newLazyPlugin(t, source)
	if err := p.use(t); err != nil {
		t.Fatal(err)
	}

	for range 2 {
		if err := p.use(t); err != nil {
			t.Fatal(err)
		}
	}

	if got := source.count(); got != 1 {
		t.Errorf("three uses downloaded %d times, want 1", got)
	}
}

func TestAUseOfAPluginAlreadyRunningForAQuestionDownloadsTheModels(t *testing.T) {
	source := newModelSource(t)
	p := newLazyPlugin(t, source)
	asked, err := p.host.Management.Activation.EnsureActiveForQuestion(t.Context(), firstBuiltInID, getTestHolder())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { p.host.Management.Activation.ReleaseQuestion(firstBuiltInID, asked.Token, getTestHolder()) })
	if got := source.count(); got != 0 {
		t.Fatalf("the question downloaded %d times, want none", got)
	}

	if err := p.use(t); err != nil {
		t.Fatal(err)
	}

	if got := source.count(); got != 1 {
		t.Errorf("the use of the running plugin downloaded %d times, want 1", got)
	}
}

func TestAFailedDownloadReturnsItsErrorAndReleasesTheLease(t *testing.T) {
	source := newModelSource(t)
	failure := errors.New("the server went away")
	source.setFailure(failure)
	p := newLazyPlugin(t, source)

	err := p.use(t)

	if !errors.Is(err, failure) {
		t.Fatalf("the use returned %v, want the download failure", err)
	}
	p.waitUntilReleased(t)
}

func TestAFailedDownloadDoesNotMarkThePluginAsFailedToStart(t *testing.T) {
	source := newModelSource(t)
	source.setFailure(errors.New("the server went away"))
	p := newLazyPlugin(t, source)

	if err := p.use(t); err == nil {
		t.Fatal("the failed download did not fail the use")
	}

	if status, ok := p.host.Management.Registry.GetPluginStatus(firstBuiltInID); !ok || status.Failed {
		t.Errorf("status = %+v, want a plugin that did not fail", status)
	}
	if items := p.host.Supervision.Health.List(); len(items) != 0 {
		t.Errorf("the failed download raised health items %+v", items)
	}
}

func TestAFailedDownloadIsTriedAgainAtTheNextUse(t *testing.T) {
	source := newModelSource(t)
	source.setFailure(errors.New("the server went away"))
	p := newLazyPlugin(t, source)
	if err := p.use(t); err == nil {
		t.Fatal("the failed download did not fail the use")
	}

	source.setFailure(nil)
	if err := p.use(t); err != nil {
		t.Fatalf("the retry failed: %v", err)
	}

	if got := source.count(); got != 2 {
		t.Errorf("the two uses downloaded %d times, want 2", got)
	}
	if _, err := os.Stat(source.path(sampleRuntime)); err != nil {
		t.Errorf("the model is missing after the retry: %v", err)
	}
}

func TestTwoFirstUsesAtOnceShareOneDownload(t *testing.T) {
	source := newModelSource(t)
	release := source.holdDownloads(t)
	p := newLazyPlugin(t, source)
	results := make(chan error, 2)
	for range 2 {
		go func() { results <- p.use(t) }()
	}
	source.waitForDownload(t)
	time.Sleep(100 * time.Millisecond)

	release()

	for range 2 {
		if err := <-results; err != nil {
			t.Errorf("a use failed: %v", err)
		}
	}
	if got := source.count(); got != 1 {
		t.Errorf("two first uses downloaded %d times, want 1", got)
	}
}

func TestTheProgressOfADownloadIsReportedAtTheFirstUseOnly(t *testing.T) {
	source := newModelSource(t)
	t.Setenv(source.envName(sampleRuntime), "")
	var mu sync.Mutex
	var seen []plugindomain.DependencyState
	registry := provision.NewObservedRegistry(func(_ catalogmanifest.DependencyRequirement, status plugindomain.DependencyStatus) {
		mu.Lock()
		defer mu.Unlock()
		seen = append(seen, status.State)
	}, source)
	p := newLazyPluginFrom(t, registry, newBuiltInManifest(firstBuiltInID, sampleRuntime))
	getSeen := func() []plugindomain.DependencyState {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(seen)
	}

	p.listTools(t)
	if got := getSeen(); len(got) != 0 {
		t.Fatalf("starting the plugin for a listing reported %v", got)
	}
	if err := p.use(t); err != nil {
		t.Fatal(err)
	}

	got := getSeen()
	if !slices.Contains(got, plugindomain.DependencyStateProvisioning) || got[len(got)-1] != plugindomain.DependencyStateReady {
		t.Errorf("the first use reported %v, want provisioning and a final ready", got)
	}
	reported := len(got)
	if err := p.use(t); err != nil {
		t.Fatal(err)
	}
	if again := getSeen(); len(again) != reported {
		t.Errorf("the second use reported %v more", again[reported:])
	}
}

func newModelManifest(dependencies string) []byte {
	return fmt.Appendf(nil, `{
		"id": %q, "name": "Example Tools", "version": "1.0.0", "capabilities": ["tools"], "targets": ["desktop.editor"],
		"permissions": [], "dependencies": [%s],
		"backend": {"entrypoint": "bin/example"}
	}`, firstBuiltInID, dependencies)
}

func newDownloadFixture(t *testing.T, data []byte, offline bool) (lazyPlugin, *atomic.Int32, string) {
	t.Helper()
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if _, err := w.Write(data); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	dir := t.TempDir()
	registry := provision.NewObservedRegistry(nil, provision.NewFileProvisioner(provision.Config{
		Dir: dir, Offline: offline, Client: server.Client(), Clock: testsupport.NewClock(),
	}))
	t.Setenv("NUMEN_DEPENDENCY_SPEECH_MODEL", "")
	manifest := newModelManifest(fmt.Sprintf(
		`{"id": "speech-model", "kind": "dataset", "version": "1", "sourceUrl": %q, "checksumSha256": %q}`,
		server.URL+"/speech.bin", sumHex(data),
	))
	return newLazyPluginFrom(t, registry, manifest), &requests, dir
}

func TestAModelIsFetchedFromItsSourceAtTheFirstUseOnly(t *testing.T) {
	data := []byte("speech model bytes of an example")
	p, requests, _ := newDownloadFixture(t, data, false)

	p.listTools(t)
	if got := requests.Load(); got != 0 {
		t.Fatalf("the listing made %d requests", got)
	}
	if err := p.use(t); err != nil {
		t.Fatal(err)
	}
	afterFirst := requests.Load()
	if afterFirst == 0 {
		t.Fatal("the first use fetched nothing")
	}
	if err := p.use(t); err != nil {
		t.Fatal(err)
	}

	if got := requests.Load(); got != afterFirst {
		t.Errorf("the second use made %d more requests", got-afterFirst)
	}
	path := p.backend.lastVars(t)["NUMEN_DEPENDENCY_SPEECH_MODEL"]
	if got, err := os.ReadFile(path); err != nil || string(got) != string(data) {
		t.Errorf("the path %q handed to the plugin holds %q, %v", path, got, err)
	}
}

func TestAnOfflineMachineFailsAtTheFirstUseNotAtTheStart(t *testing.T) {
	p, requests, _ := newDownloadFixture(t, []byte("speech model bytes of an example"), true)

	if names := p.listTools(t); !slices.Equal(names, []string{lazyToolName}) {
		t.Fatalf("the listing gave %v, want the tools of a plugin whose model is not there", names)
	}
	err := p.use(t)

	if !errors.Is(err, provision.ErrOffline) {
		t.Errorf("the first use returned %v, want ErrOffline", err)
	}
	if got := requests.Load(); got != 0 {
		t.Errorf("an offline machine made %d requests", got)
	}
}

func TestAModelWithAWrongChecksumFailsAtTheFirstUse(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if _, err := w.Write([]byte("tampered bytes")); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	registry := provision.NewObservedRegistry(nil, provision.NewFileProvisioner(provision.Config{
		Dir: t.TempDir(), Client: server.Client(), Clock: testsupport.NewClock(),
	}))
	t.Setenv("NUMEN_DEPENDENCY_SPEECH_MODEL", "")
	p := newLazyPluginFrom(t, registry, newModelManifest(fmt.Sprintf(
		`{"id": "speech-model", "kind": "dataset", "version": "1", "sourceUrl": %q, "checksumSha256": %q}`,
		server.URL+"/speech.bin", sumHex([]byte("the real bytes")),
	)))

	if names := p.listTools(t); !slices.Equal(names, []string{lazyToolName}) {
		t.Fatalf("the listing gave %v", names)
	}
	err := p.use(t)

	if !errors.Is(err, provision.ErrChecksum) {
		t.Errorf("the first use returned %v, want ErrChecksum", err)
	}
}

func TestAModelOfAnotherPlatformFailsAtTheFirstUse(t *testing.T) {
	registry := provision.NewObservedRegistry(nil, provision.NewFileProvisioner(provision.Config{Dir: t.TempDir(), Clock: testsupport.NewClock()}))
	t.Setenv("NUMEN_DEPENDENCY_SPEECH_MODEL", "")
	p := newLazyPluginFrom(t, registry, newModelManifest(fmt.Sprintf(
		`{"id": "speech-model", "kind": "dataset", "version": "1", "sources": [{"platform": "plan9/mips64", "sourceUrl": "https://models.invalid/speech.bin", "checksumSha256": %q}]}`,
		sumHex([]byte("x")),
	)))

	if names := p.listTools(t); !slices.Equal(names, []string{lazyToolName}) {
		t.Fatalf("the listing gave %v", names)
	}
	err := p.use(t)

	if err == nil || !strings.Contains(err.Error(), "plan9") && !strings.Contains(err.Error(), "no source") {
		t.Errorf("the first use returned %v, want the missing source for this platform", err)
	}
}

func TestAPluginDeclaringARuntimeNothingProvisionsIsNotStartedForAQuestion(t *testing.T) {
	t.Setenv("NUMEN_DEPENDENCY_SAMPLE_RUNTIME", "")
	p := newLazyPluginFrom(t, provision.NewObservedRegistry(nil), newBuiltInManifest(firstBuiltInID, sampleRuntime))

	_, err := p.host.Management.Activation.EnsureActiveForQuestion(t.Context(), firstBuiltInID, getTestHolder())

	if !errors.Is(err, supervision.ErrRuntimeUnavailable) {
		t.Errorf("EnsureActiveForQuestion() = %v, want ErrRuntimeUnavailable at launch", err)
	}
}

func newModelSettingManifest() []byte {
	return fmt.Appendf(nil, `{
		"id": %q, "name": "Example Tools", "version": "1.0.0", "capabilities": ["tools"], "targets": ["desktop.editor"],
		"permissions": [],
		"dependencies": [{"id": "model-small", "kind": "runtime"}, {"id": "model-large", "kind": "runtime"}],
		"backend": {"entrypoint": "bin/example"},
		"settings": [{"key": "model", "type": "enum", "title": "Model", "default": "small", "options": [
			{"value": "small", "title": "Small", "dependencies": ["model-small"]},
			{"value": "large", "title": "Large", "dependencies": ["model-large"]}
		]}]
	}`, firstBuiltInID)
}

func TestAChangedModelSettingDownloadsTheNewModelAtTheNextUse(t *testing.T) {
	source := newModelSource(t)
	t.Setenv(source.envName("model-small"), "")
	t.Setenv(source.envName("model-large"), "")
	p := newLazyPluginFrom(t, provision.NewObservedRegistry(nil, source), newModelSettingManifest())
	if err := p.use(t); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(source.path("model-small")); err != nil {
		t.Fatalf("the small model is not there after the first use: %v", err)
	}

	if err := updateSetting(t.Context(), p.host, firstBuiltInID, "model", "large"); err != nil {
		t.Fatal(err)
	}
	if got := source.count(); got != 1 {
		t.Fatalf("changing the setting downloaded %d times in total, want only the first use", got)
	}
	if err := p.use(t); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(source.path("model-large")); err != nil {
		t.Errorf("the large model is not there after the next use: %v", err)
	}
	if got := source.count(); got != 2 {
		t.Errorf("downloaded %d times in total, want 2", got)
	}
}

func newEventPlugin(t *testing.T, source *modelSource) (*pluginhost.Host, *eventRecorder) {
	t.Helper()
	t.Setenv(source.envName(sampleRuntime), "")
	host := newBuiltInHost(t)
	host.Supervision.Dependencies.Replace(provision.NewObservedRegistry(nil, source))
	recorder := &eventRecorder{}
	manifest := fmt.Appendf(nil, `{
		"id": %q, "name": "Example Events", "version": "1.0.0", "targets": ["desktop.editor"], "permissions": [],
		"activationEvents": [%q], "dependencies": [{"id": %q, "kind": "runtime"}],
		"backend": {"entrypoint": "bin/example"}
	}`, firstBuiltInID, catalogmanifest.ActivationEventIndexChanged, sampleRuntime)
	registerBuiltIn(t, host.Management, manifest, func(map[string]string) (http.Handler, error) {
		mux := http.NewServeMux()
		wiretest.HandlePluginLifecycle(mux)
		mux.Handle(numenv1connect.NewPluginEventServiceHandler(recorder))
		return mux, nil
	})
	return host, recorder
}

func newSpawnedLazyHost(t *testing.T, source *modelSource) *pluginhost.Host {
	t.Helper()
	host := newEventHost(t,
		map[string]catalogmanifest.Manifest{
			"md.test.lazy": {Dependencies: []catalogmanifest.DependencyRequirement{{ID: sampleRuntime, Kind: catalogmanifest.DependencyKindRuntime}}},
		},
		map[string]int{"md.test.lazy": servePlugin(t, nil)},
	)
	host.Supervision.Dependencies.Replace(provision.NewObservedRegistry(nil, source))
	return host
}

func TestAnEventDeliveredToAPluginDownloadsNothing(t *testing.T) {
	source := newModelSource(t)
	host, recorder := newEventPlugin(t, source)

	activated, err := host.Management.Events.TriggerEvent(t.Context(), indexChangedEvent)

	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(activated, []string{firstBuiltInID}) || !slices.Equal(recorder.getEvents(), []string{indexChangedEvent}) {
		t.Fatalf("activated %v and delivered %v, want the plugin to get the event", activated, recorder.getEvents())
	}
	if got := source.count(); got != 0 {
		t.Errorf("the event downloaded %d times, want none", got)
	}
}

func TestAnEventOnTheBusDownloadsNothing(t *testing.T) {
	source := newModelSource(t)
	host, recorder := newEventPlugin(t, source)

	host.Management.Events.EmitEvent(indexChangedEvent)
	if err := host.Management.Events.GetEventBus().Flush(t.Context()); err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(recorder.getEvents(), []string{indexChangedEvent}) {
		t.Fatalf("delivered %v", recorder.getEvents())
	}
	if got := source.count(); got != 0 {
		t.Errorf("the event downloaded %d times, want none", got)
	}
}

func TestASpawnedPluginDownloadsOnceForTwoUses(t *testing.T) {
	source := newModelSource(t)
	host := newSpawnedLazyHost(t, source)

	for range 2 {
		endpoint, err := host.Management.Activation.EnsureActive(t.Context(), "md.test.lazy", getTestHolder())
		if err != nil {
			t.Fatal(err)
		}
		host.Management.Activation.Release("md.test.lazy", endpoint.Token, getTestHolder())
	}

	if got := source.count(); got != 1 {
		t.Errorf("two uses downloaded %d times, want 1", got)
	}
}

func TestADownloadLongerThanTheActivationTimeoutStillFinishes(t *testing.T) {
	source := newModelSource(t)
	host := newSpawnedLazyHost(t, source)
	policy := lifecycle.DefaultActivationPolicy()
	policy.Timeout = 50 * time.Millisecond
	host.Supervision.Processes.SetActivationPolicy(policy)
	release := source.holdDownloads(t)
	done := make(chan error, 1)

	go func() {
		endpoint, err := host.Management.Activation.EnsureActive(t.Context(), "md.test.lazy", getTestHolder())
		if err == nil {
			host.Management.Activation.Release("md.test.lazy", endpoint.Token, getTestHolder())
		}
		done <- err
	}()
	source.waitForDownload(t)
	time.Sleep(4 * policy.Timeout)
	release()

	if err := <-done; err != nil {
		t.Errorf("the use failed after a download longer than the activation timeout: %v", err)
	}
	if status, ok := host.Management.Registry.GetPluginStatus("md.test.lazy"); !ok || status.Failed {
		t.Errorf("status = %+v, want a plugin that did not fail", status)
	}
}

func TestAListenerConnectionDownloadsNothing(t *testing.T) {
	source := newModelSource(t)
	root := t.TempDir()
	port := getFreePort(t)
	createListenerPlugin(t, root, "mcp", port)
	manifestPath := filepath.Join(root, "relay", "manifest.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	declared := strings.Replace(string(raw), `"backend": {"entrypoint": "bin/relay"},`,
		`"dependencies": [{"id": "sample-runtime", "kind": "runtime"}], "backend": {"entrypoint": "bin/relay"},`, 1)
	if declared == string(raw) {
		t.Fatal("the listener manifest has no backend line to put the dependency before")
	}
	if err := os.WriteFile(manifestPath, []byte(declared), 0o600); err != nil {
		t.Fatal(err)
	}
	host := pluginhost.New(testsupport.GetMachine(), "desktop.editor", root)
	teardownAfter(t, host.Management)
	host.Supervision.Dependencies.Replace(provision.NewObservedRegistry(nil, source))
	var started atomic.Int32
	host.Supervision.Processes.SetLauncher(startRelayLauncher(t, &started, "listener.mcp="+strconv.Itoa(serveEcho(t))))
	discover(t, host)

	conn := connectListener(t, port)
	if _, err := io.WriteString(conn, "hello\n"); err != nil {
		t.Fatal(err)
	}
	reply, err := bufio.NewReader(conn).ReadString('\n')

	if err != nil || reply != "hello\n" {
		t.Fatalf("the listener answered %q, %v", reply, err)
	}
	if got := source.count(); got != 0 {
		t.Errorf("the connection downloaded %d times, want none", got)
	}
}

func sumHex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func TestCancelingTheDependenciesStopsTheDownloadInProgress(t *testing.T) {
	source := newModelSource(t)
	source.holdDownloads(t)
	p := newLazyPlugin(t, source)
	p.host.Supervision.Services.ObserveDependencies(func(catalogmanifest.DependencyRequirement, plugindomain.DependencyStatus) {})
	answer := make(chan error, 1)
	go func() { answer <- p.use(t) }()
	source.waitForDownload(t)

	p.host.Supervision.Services.CancelDependencies()
	if err := <-answer; !errors.Is(err, context.Canceled) {
		t.Errorf("the use received %v, want the cancellation", err)
	}
}

func TestCancelingTheDependenciesStopsEveryVersionOfOneDependency(t *testing.T) {
	source := newModelSource(t)
	source.holdDownloads(t)
	host := newBuiltInHost(t)
	registry := provision.NewObservedRegistry(host.Supervision.Services.Observe, source)
	host.Supervision.Dependencies.Replace(registry)
	p := newLazyPluginOn(t, host, newBuiltInManifest(firstBuiltInID, sampleRuntime))
	p.host.Supervision.Services.ObserveDependencies(func(catalogmanifest.DependencyRequirement, plugindomain.DependencyStatus) {})
	older := catalogmanifest.DependencyRequirement{ID: sampleRuntime, Kind: catalogmanifest.DependencyKindModel, Version: "1"}
	newer := catalogmanifest.DependencyRequirement{ID: sampleRuntime, Kind: catalogmanifest.DependencyKindModel, Version: "2"}
	answers := make(chan error, 2)
	for _, req := range []catalogmanifest.DependencyRequirement{older, newer} {
		go func() {
			_, err := registry.Provision(t.Context(), req, nil)
			answers <- err
		}()
	}
	source.waitForDownload(t)
	source.waitForDownload(t)
	if !registry.Cancel(older) {
		t.Fatal("Cancel found no download of the older version")
	}
	if err := <-answers; !errors.Is(err, context.Canceled) {
		t.Fatalf("the older version received %v, want the cancellation", err)
	}

	p.host.Supervision.Services.CancelDependencies()
	if err := <-answers; !errors.Is(err, context.Canceled) {
		t.Errorf("the newer version received %v, want the cancellation", err)
	}
}
