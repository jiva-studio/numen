package plugin_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/process"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const (
	firstBuiltInID  = "md.example.first"
	secondBuiltInID = "md.example.second"
)

func newBuiltInManifest(id string, runtimes ...string) []byte {
	declared := ""
	if len(runtimes) > 0 {
		dependencies := make([]string, 0, len(runtimes))
		for _, runtime := range runtimes {
			dependencies = append(dependencies, fmt.Sprintf(`{"id": %q, "kind": "runtime"}`, runtime))
		}
		declared = fmt.Sprintf(`"dependencies": [%s],`, strings.Join(dependencies, ","))
	}
	return fmt.Appendf(nil, `{
		"id": %q,
		"name": "Example Tools",
		"version": "1.0.0",
		"capabilities": ["tools"],
		"targets": ["desktop.editor"],
		"permissions": [],
		%s
		"services": ["numen.v1.PluginToolService"], "backend": {"entrypoint": "bin/example"},
		"settings": [{"key": "model", "type": "string", "title": "Model", "default": "small-model"}]
	}`, id, declared)
}

// titledTools lists one tool, titled with the text it was made with.
type titledTools struct {
	numenv1connect.UnimplementedPluginToolServiceHandler
	title string
}

func (a *titledTools) ListTools(
	_ context.Context, _ *connect.Request[v1.ListToolsRequest],
) (*connect.Response[v1.ListToolsResponse], error) {
	return connect.NewResponse(&v1.ListToolsResponse{Tools: []*v1.PluginTool{{Name: "note_read", Title: a.title}}}), nil
}

// builtInBackend is the backend of one built-in plugin: the variables it was made with and the
// Configure calls it received.
type builtInBackend struct {
	mu    sync.Mutex
	vars  []map[string]string
	probe *configured
}

func (b *builtInBackend) serve(title string) func(map[string]string) (http.Handler, error) {
	return func(vars map[string]string) (http.Handler, error) {
		b.mu.Lock()
		b.vars = append(b.vars, vars)
		b.mu.Unlock()
		mux := http.NewServeMux()
		mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(b.probe))
		mux.Handle(numenv1connect.NewPluginToolServiceHandler(&titledTools{title: title}))
		return mux, nil
	}
}

func (b *builtInBackend) lastVars(t *testing.T) map[string]string {
	t.Helper()
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.vars) == 0 {
		t.Fatal("the built-in plugin was never served")
	}
	return b.vars[len(b.vars)-1]
}

func newBuiltInBackend() *builtInBackend {
	return &builtInBackend{probe: &configured{}}
}

func newBuiltInHost(t *testing.T, dirs ...string) *container.PluginRoles {
	t.Helper()
	host := container.NewPluginRoles(testsupport.GetMachine(), "desktop.editor", dirs...)
	t.Cleanup(func() {
		if err := host.Teardown.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond); err != nil {
			t.Error(err)
		}
	})
	return host
}

func registerBuiltIn(t *testing.T, host *container.PluginRoles, manifest []byte, serve func(map[string]string) (http.Handler, error)) {
	t.Helper()
	if err := host.Discovery.RegisterBuiltIn(manifest, serve); err != nil {
		t.Fatalf("RegisterBuiltIn: %v", err)
	}
}

func getToolTitle(t *testing.T, host *container.PluginRoles, endpoint transport.Endpoint) string {
	t.Helper()
	client := numenv1connect.NewPluginToolServiceClient(host.Activation.ClientLookup().GetClient(endpoint), transport.GetBaseURL(endpoint))
	res, err := client.ListTools(t.Context(), connect.NewRequest(&v1.ListToolsRequest{}))
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	return res.Msg.GetTools()[0].GetTitle()
}

// Two built-in plugins serving the same service each answer from their own handler.
func TestEachBuiltInPluginAnswersFromItsOwnHandler(t *testing.T) {
	host := newBuiltInHost(t)
	registerBuiltIn(t, host, newBuiltInManifest(firstBuiltInID), newBuiltInBackend().serve("first tools"))
	registerBuiltIn(t, host, newBuiltInManifest(secondBuiltInID), newBuiltInBackend().serve("second tools"))

	for id, want := range map[string]string{firstBuiltInID: "first tools", secondBuiltInID: "second tools"} {
		endpoint, err := host.Activation.EnsureActive(t.Context(), id)
		if err != nil {
			t.Fatalf("EnsureActive(%s): %v", id, err)
		}
		if endpoint.Port != 0 || endpoint.Host != "inprocess" {
			t.Fatalf("%s is not served in-process: %+v", id, endpoint)
		}
		if got := getToolTitle(t, host, endpoint); got != want {
			t.Errorf("%s answered %q, want %q", id, got, want)
		}
	}
}

// A built-in plugin is registered without being found on disk, so a capability finds it.
func TestABuiltInPluginIsFoundByItsCapability(t *testing.T) {
	host := newBuiltInHost(t)
	registerBuiltIn(t, host, newBuiltInManifest(firstBuiltInID), newBuiltInBackend().serve("first tools"))

	found, ok := host.Registry.FindPluginByCapability("tools")
	if !ok || found.Manifest.ID != firstBuiltInID {
		t.Fatalf("FindPluginByCapability(tools) = %+v, %v", found, ok)
	}
}

// A built-in plugin is made with the variables a launched process is given, and the token it is
// given names it to the host.
func TestABuiltInPluginIsGivenItsOwnLaunchVariables(t *testing.T) {
	host := newBuiltInHost(t)
	host.Mode.SetVaultPath("/vaults/example")
	backend := newBuiltInBackend()
	registerBuiltIn(t, host, newBuiltInManifest(firstBuiltInID), backend.serve("first tools"))

	endpoint, err := host.Activation.EnsureActive(t.Context(), firstBuiltInID)
	if err != nil {
		t.Fatalf("EnsureActive: %v", err)
	}
	vars := backend.lastVars(t)
	if vars["NUMEN_PLUGIN_ID"] != firstBuiltInID || vars["NUMEN_VAULT_PATH"] != "/vaults/example" {
		t.Errorf("vars = %v", vars)
	}
	if vars["NUMEN_PLUGIN_TOKEN"] == "" || vars["NUMEN_PLUGIN_TOKEN"] != endpoint.Token {
		t.Errorf("token %q, endpoint token %q", vars["NUMEN_PLUGIN_TOKEN"], endpoint.Token)
	}
	if id, ok := host.Activation.GetPluginIDByToken(endpoint.Token); !ok || id != firstBuiltInID {
		t.Errorf("GetPluginIDByToken = %q, %v", id, ok)
	}
}

// Configure calls the built-in plugin's own lifecycle service with its settings.
func TestABuiltInPluginIsConfiguredWithItsSettings(t *testing.T) {
	host := newBuiltInHost(t)
	host.Settings.SetSettings(map[string]map[string]any{firstBuiltInID: {"model": "large-model"}})
	backend := newBuiltInBackend()
	registerBuiltIn(t, host, newBuiltInManifest(firstBuiltInID), backend.serve("first tools"))

	if _, err := host.Activation.EnsureActive(t.Context(), firstBuiltInID); err != nil {
		t.Fatalf("EnsureActive: %v", err)
	}
	call := backend.probe.last(t)
	if call.GetValues()["model"] != "large-model" {
		t.Errorf("Configure values = %v", call.GetValues())
	}
}

// A backend that cannot be made fails the activation with the plugin named.
func TestABuiltInPluginWhoseBackendFailsIsNotActivated(t *testing.T) {
	host := newBuiltInHost(t)
	failure := errors.New("the backend cannot start")
	registerBuiltIn(t, host, newBuiltInManifest(firstBuiltInID), func(map[string]string) (http.Handler, error) {
		return nil, failure
	})

	_, err := host.Activation.EnsureActive(t.Context(), firstBuiltInID)
	if !errors.Is(err, failure) || !strings.Contains(err.Error(), firstBuiltInID) {
		t.Fatalf("EnsureActive() = %v, want the failure with the plugin named", err)
	}
}

// A built-in plugin is registered once, and only for the target it names.
func TestABuiltInPluginIsRefusedTwiceAndForAnotherTarget(t *testing.T) {
	host := newBuiltInHost(t)
	serve := newBuiltInBackend().serve("first tools")
	registerBuiltIn(t, host, newBuiltInManifest(firstBuiltInID), serve)

	if err := host.Discovery.RegisterBuiltIn(newBuiltInManifest(firstBuiltInID), serve); !errors.Is(err, plugin.ErrBuiltInRegistered) {
		t.Errorf("second registration = %v, want plugin.ErrBuiltInRegistered", err)
	}
	other := container.NewPluginRoles(testsupport.GetMachine(), "mobile.reader")
	t.Cleanup(func() {
		if err := other.Teardown.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond); err != nil {
			t.Error(err)
		}
	})
	if err := other.Discovery.RegisterBuiltIn(newBuiltInManifest(firstBuiltInID), serve); !errors.Is(err, plugin.ErrNotForTarget) {
		t.Errorf("registration for another target = %v, want plugin.ErrNotForTarget", err)
	}
	if err := host.Discovery.RegisterBuiltIn([]byte(`{"id": "not a manifest"`), serve); err == nil {
		t.Error("a manifest that does not parse is registered")
	}
}

// A plugin found on disk with no built-in of its id is launched as a process.
func TestAPluginOnDiskThatIsNotBuiltInIsLaunched(t *testing.T) {
	tmp := t.TempDir()
	wiretest.CreateTestPluginTree(t, tmp)
	host := newBuiltInHost(t, tmp)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatalf("Discover: %v", err)
	}
	launched := make(chan string, 1)
	host.Processes.SetLauncher(func(_ context.Context, p manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		launched <- p.Manifest.ID
		return nil, nil, errors.New("no process in this test")
	})

	if _, err := host.Activation.EnsureActive(t.Context(), "org.example.sample"); err == nil {
		t.Fatal("the launch that failed activated the plugin")
	}
	if got := <-launched; got != "org.example.sample" {
		t.Fatalf("launched %q", got)
	}
}

func TestABuiltInPluginRemovedFromDiskIsTheBuiltInAgain(t *testing.T) {
	tmp := t.TempDir()
	wiretest.CreateTestPluginTree(t, tmp)
	host := newBuiltInHost(t, tmp)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatalf("Discover: %v", err)
	}
	manifest, err := os.ReadFile(filepath.Join(tmp, "sample", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	registerBuiltIn(t, host, manifest, newBuiltInBackend().serve("built-in sample"))
	if p, _ := host.Registry.GetPlugin("org.example.sample"); p.Dir == "" {
		t.Fatal("the copy on disk was replaced by the built-in")
	}

	if err := os.RemoveAll(filepath.Join(tmp, "sample")); err != nil {
		t.Fatal(err)
	}
	if err := host.Discovery.Discover(); err != nil {
		t.Fatalf("Discover: %v", err)
	}

	p, ok := host.Registry.GetPlugin("org.example.sample")
	if !ok || p.Dir != "" {
		t.Fatalf("GetPlugin = %+v, %v; want the built-in", p, ok)
	}
	endpoint, err := host.Activation.EnsureActive(t.Context(), "org.example.sample")
	if err != nil {
		t.Fatalf("EnsureActive: %v", err)
	}
	if got := getToolTitle(t, host, endpoint); got != "built-in sample" {
		t.Fatalf("answered %q", got)
	}
}

func newFixedCatalog(library, provider string, err error) *fakeCatalog {
	catalog := newONNXCatalog()
	catalog.resolved[plugindomain.RuntimeONNX] = plugindomain.ResolvedDependency{
		ID:      plugindomain.RuntimeONNX,
		EnvVars: map[string]string{process.EnvONNXRuntimeLib: library, process.EnvONNXProvider: provider},
	}
	catalog.err = err
	return catalog
}

// A built-in plugin that declares the runtime finds the same variables a launched process is
// given, in the process that is its own and among the variables it is made with.
func TestABuiltInPluginIsGivenTheRuntime(t *testing.T) {
	t.Setenv(process.EnvONNXRuntimeLib, "")
	t.Setenv(process.EnvONNXProvider, "")
	host := newBuiltInHost(t)
	host.Services.SetDependencies(newFixedCatalog("/models/libonnxruntime.so", "coreml", nil))
	backend := newBuiltInBackend()
	registerBuiltIn(t, host, newBuiltInManifest(firstBuiltInID, "onnxruntime"), backend.serve("first tools"))

	if _, err := host.Activation.EnsureActive(t.Context(), firstBuiltInID); err != nil {
		t.Fatalf("EnsureActive: %v", err)
	}
	if got := os.Getenv(process.EnvONNXRuntimeLib); got != "/models/libonnxruntime.so" {
		t.Errorf("%s = %q", process.EnvONNXRuntimeLib, got)
	}
	if got := os.Getenv(process.EnvONNXProvider); got != "coreml" {
		t.Errorf("%s = %q", process.EnvONNXProvider, got)
	}
	if got := backend.lastVars(t)[process.EnvONNXProvider]; got != "coreml" {
		t.Errorf("vars hold %s = %q", process.EnvONNXProvider, got)
	}
}

// A built-in plugin whose runtime cannot be provisioned is not activated.
func TestABuiltInPluginIsNotActivatedWithoutItsRuntime(t *testing.T) {
	t.Setenv(process.EnvONNXRuntimeLib, "")
	failure := errors.New("the archive is not the pinned one")
	host := newBuiltInHost(t)
	host.Services.SetDependencies(newFixedCatalog("", "", failure))
	registerBuiltIn(t, host, newBuiltInManifest(firstBuiltInID, "onnxruntime"), newBuiltInBackend().serve("first tools"))

	_, err := host.Activation.EnsureActive(t.Context(), firstBuiltInID)
	if !errors.Is(err, failure) || !strings.Contains(err.Error(), firstBuiltInID) {
		t.Fatalf("EnsureActive() = %v, want the failure with the plugin named", err)
	}
	if got := os.Getenv(process.EnvONNXRuntimeLib); got != "" {
		t.Errorf("%s = %q after a failed provisioning", process.EnvONNXRuntimeLib, got)
	}
}

// An activation that lost the race with a shutdown leaves the plugin dormant and gives no endpoint.
func TestABuiltInActivationThatLostTheRaceWithShutdownIsRejected(t *testing.T) {
	host := newBuiltInHost(t)
	registerBuiltIn(t, host, newBuiltInManifest(firstBuiltInID), newBuiltInBackend().serve("first tools"))
	p, _ := host.Registry.GetPlugin(firstBuiltInID)
	sup := supervisorOf(host, firstBuiltInID)

	endpoint, err := host.Processes.Start(t.Context(), plugin.GetLaunch(host.Registry, p), sup)

	if !errors.Is(err, lifecycle.ErrActivationAborted) {
		t.Fatalf("expected errActivationAborted, got %v", err)
	}
	if !supervision.IsNoEndpoint(endpoint) {
		t.Fatalf("expected no endpoint, got %+v", endpoint)
	}
	if sup.GetState() != lifecycle.StateDormant {
		t.Fatalf("expected StateDormant, got %v", sup.GetState())
	}
}

type newerProtocolLifecycle struct {
	numenv1connect.UnimplementedPluginLifecycleServiceHandler
}

func (newerProtocolLifecycle) Ping(
	_ context.Context,
	_ *connect.Request[v1.PingRequest],
) (*connect.Response[v1.PingResponse], error) {
	return connect.NewResponse(&v1.PingResponse{Status: "ok", Protocol: manifest.ProtocolVersion + 1}), nil
}

func TestABuiltInPluginThatUsesAnotherProtocolIsRefused(t *testing.T) {
	host := newBuiltInHost(t)
	registerBuiltIn(t, host, newBuiltInManifest(firstBuiltInID), func(map[string]string) (http.Handler, error) {
		mux := http.NewServeMux()
		mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(newerProtocolLifecycle{}))
		return mux, nil
	})

	_, err := host.Activation.EnsureActive(t.Context(), firstBuiltInID)
	if !errors.Is(err, manifest.ErrUnsupportedProtocol) || !strings.Contains(err.Error(), firstBuiltInID+" uses protocol 2") {
		t.Fatalf("EnsureActive() = %v, want %v naming the plugin", err, manifest.ErrUnsupportedProtocol)
	}
}

// A handler that flushes is heard before it ends.
func TestAFlushSendsTheHeaderOfAStreamingHandler(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	transport := transport.NewInProcessTransport(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Error(err)
		}
		<-release
	}))
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://inprocess/", nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Error(err)
	}
}

func TestACallWhoseContextEndsAfterTheHeaderEndsWithThatError(t *testing.T) {
	transport := transport.NewInProcessTransport(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		<-r.Context().Done()
	}))
	ctx, cancel := context.WithCancel(t.Context())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://inprocess/", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Error(err)
		}
	}()

	cancel()
	read := make(chan error, 1)
	go func() {
		_, err := io.ReadAll(resp.Body)
		read <- err
	}()

	select {
	case err := <-read:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("the body ended with %v, want the cancellation", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the body was not ended by the cancellation")
	}
}

func TestABuiltInHandlerWhoseActivationFailsIsShutDown(t *testing.T) {
	host := newBuiltInHost(t)
	backend := newBuiltInBackend()
	registerBuiltIn(t, host, newBuiltInManifest(firstBuiltInID), backend.serve("first tools"))
	p, _ := host.Registry.GetPlugin(firstBuiltInID)
	sup := supervisorOf(host, firstBuiltInID)

	if _, err := host.Processes.Start(t.Context(), plugin.GetLaunch(host.Registry, p), sup); !errors.Is(err, lifecycle.ErrActivationAborted) {
		t.Fatalf("activatePlugin = %v, want errActivationAborted", err)
	}

	backend.probe.mu.Lock()
	defer backend.probe.mu.Unlock()
	if backend.probe.shutdowns != 1 {
		t.Fatalf("the handler was shut down %d times, want once", backend.probe.shutdowns)
	}
}

// A built-in plugin's permissions are the built-in's again when its copy on disk leaves.
func TestInProcess_RemovedBuiltInKeepsOwnPermissions(t *testing.T) {
	tmp := t.TempDir()
	wiretest.CreateTestPluginTree(t, tmp)
	host := newBuiltInHost(t, tmp)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatalf("Discover: %v", err)
	}
	manifestJSON, err := os.ReadFile(filepath.Join(tmp, "sample", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	builtIn := strings.Replace(string(manifestJSON), `"permissions": ["vault:read"]`, `"permissions": ["vault:write"]`, 1)
	if builtIn == string(manifestJSON) {
		t.Fatal("the fixture's permissions are not the ones replaced")
	}
	registerBuiltIn(t, host, []byte(builtIn), newBuiltInBackend().serve("built-in sample"))

	if err := os.RemoveAll(filepath.Join(tmp, "sample")); err != nil {
		t.Fatal(err)
	}
	if err := host.Discovery.Discover(); err != nil {
		t.Fatalf("Discover: %v", err)
	}

	if err := host.Activation.CheckPluginPermission("org.example.sample", manifest.PermissionVaultWrite); err != nil {
		t.Errorf("the built-in's own permission: %v", err)
	}
	if err := host.Activation.CheckPluginPermission("org.example.sample", manifest.PermissionVaultRead); err == nil {
		t.Error("the permission of the copy that left is still granted")
	}
}

// A copy on disk is launched as a process, whatever built-in shares its id.
func TestAPluginOnDiskIsLaunchedWhereABuiltInSharesItsId(t *testing.T) {
	tmp := t.TempDir()
	wiretest.CreateTestPluginTree(t, tmp)
	host := newBuiltInHost(t, tmp)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatalf("Discover: %v", err)
	}
	manifest, err := os.ReadFile(filepath.Join(tmp, "sample", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	registerBuiltIn(t, host, manifest, newBuiltInBackend().serve("built-in sample"))

	endpoint, err := host.Activation.EnsureActive(t.Context(), "org.example.sample")

	if err == nil {
		t.Fatalf("the copy on disk was served by the built-in: %+v", endpoint)
	}
}

// A handler that cannot be shut down is reported, and the activation error stays the cause.
func TestAShutdownThatFailsOnAnUnactivatedBuiltInIsReported(t *testing.T) {
	host := newBuiltInHost(t)
	reported := make(chan error, 4)
	host.Failures.SetErrorHandler(func(err error) { reported <- err })
	registerBuiltIn(t, host, newBuiltInManifest(firstBuiltInID), func(map[string]string) (http.Handler, error) {
		mux := http.NewServeMux()
		mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(newerProtocolLifecycle{}))
		return mux, nil
	})

	_, err := host.Activation.EnsureActive(t.Context(), firstBuiltInID)

	if !errors.Is(err, manifest.ErrUnsupportedProtocol) {
		t.Fatalf("EnsureActive() = %v, want %v", err, manifest.ErrUnsupportedProtocol)
	}
	select {
	case got := <-reported:
		if !strings.Contains(got.Error(), "shut down the handler") {
			t.Errorf("reported %v", got)
		}
	default:
		t.Error("the failed shutdown was not reported")
	}
}

// A built-in plugin is served the host variables a launched process is given and no others.
func TestABuiltInPluginIsNotGivenTheVariablesOfTheHostProcess(t *testing.T) {
	t.Setenv("DESKTOP_ONLY_SETTING", "excluded")
	t.Setenv("PATH", "/usr/bin")
	t.Setenv("DESKTOP_API_KEY", "secret")
	host := newBuiltInHost(t)
	backend := newBuiltInBackend()
	registerBuiltIn(t, host, newBuiltInManifest(firstBuiltInID), backend.serve("first tools"))

	if _, err := host.Activation.EnsureActive(t.Context(), firstBuiltInID); err != nil {
		t.Fatalf("EnsureActive: %v", err)
	}

	vars := backend.lastVars(t)
	if vars["PATH"] != "/usr/bin" {
		t.Errorf("PATH = %q, want the host's", vars["PATH"])
	}
	for _, name := range []string{"DESKTOP_ONLY_SETTING", "DESKTOP_API_KEY"} {
		if _, isSet := vars[name]; isSet {
			t.Errorf("the plugin was given %s", name)
		}
	}
}
