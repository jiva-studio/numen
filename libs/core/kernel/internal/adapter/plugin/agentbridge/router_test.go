package agentbridge_test

import (
	"context"
	"fmt"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/agentbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const (
	disabledProvider  = "org.example.agents.disabled"
	failingProvider   = "org.example.agents.failing"
	answeringProvider = "org.example.agents.answering"
)

// answeringServer lists the one program "lark".
type answeringServer struct {
	numenv1connect.UnimplementedAgentServiceHandler
}

func (answeringServer) GetAgentPrograms(
	context.Context, *connect.Request[v1.GetAgentProgramsRequest],
) (*connect.Response[v1.GetAgentProgramsResponse], error) {
	return connect.NewResponse(&v1.GetAgentProgramsResponse{Programs: []*v1.AgentProgram{{
		Name: "lark", Title: "Lark", IsInstalled: true,
	}}}), nil
}

func writeAgentProvider(t *testing.T, dir, id string) {
	t.Helper()
	pluginDir := filepath.Join(dir, id)
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	text := fmt.Sprintf(`{
		"id": %q, "name": %q, "version": "1.0.0", "description": "Agent provider",
		"capabilities": ["ai.agent.provider"], "targets": ["desktop.editor"],
		"permissions": [],
		"services": ["numen.v1.AgentService"], "backend": {"entrypoint": "bin/agent"}
	}`, id, id)
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

// providerHost is a plugin host whose agent provider plugins are served by the handlers given, by
// plugin id, with the plugins of the ids in disabled switched off. It records the plugins it launches.
type providerHost struct {
	pluginhost.Roles

	mu       sync.Mutex
	launched []string
}

func (h *providerHost) getLaunched() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.launched...)
}

func newProviderHost(t *testing.T, handlers map[string]numenv1connect.AgentServiceHandler, disabled ...string) *providerHost {
	t.Helper()
	dir := t.TempDir()
	ports := map[string]int{}
	for id, handler := range handlers {
		writeAgentProvider(t, dir, id)
		mux := http.NewServeMux()
		wiretest.HandlePluginLifecycle(mux)
		mux.Handle(numenv1connect.NewAgentServiceHandler(handler))
		server := httptest.NewServer(mux)
		t.Cleanup(server.Close)
		ports[id] = wiretest.GetPort(t, server)
	}

	inner := pluginhost.New(testsupport.GetMachine(), "desktop.editor", dir)
	host := &providerHost{Roles: pluginhost.Join(inner)}
	settings := map[string]map[string]any{}
	for _, id := range disabled {
		settings[id] = map[string]any{"enabled": false}
	}
	inner.Settings.SetSettings(settings)
	if err := inner.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}
	processes := map[string]*os.Process{}
	for id := range handlers {
		processes[id] = wiretest.StartSleepProcess(t).Process
	}
	inner.Processes.SetLauncher(func(_ context.Context, p manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		host.mu.Lock()
		host.launched = append(host.launched, p.Manifest.ID)
		host.mu.Unlock()
		ready := strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", ports[p.Manifest.ID]))
		return processes[p.Manifest.ID], io.NopCloser(ready), nil
	})
	return host
}

func TestTheProgramsListedAreThoseOfTheEnabledProvidersThatAnswer(t *testing.T) {
	host := newProviderHost(t, map[string]numenv1connect.AgentServiceHandler{
		disabledProvider:  answeringServer{},
		failingProvider:   &floodAgentServer{},
		answeringProvider: answeringServer{},
	}, disabledProvider)

	programs, err := agentbridge.NewRouter(host).GetPrograms(t.Context(), nil)

	if err == nil {
		t.Error("the failing provider was not reported")
	}
	if len(programs) != 1 || programs[0].Provider != answeringProvider || programs[0].Name != "lark" {
		t.Errorf("listed %+v, want only lark of %s", programs, answeringProvider)
	}
}

func TestListingReportsTheEnabledProviderThatFailsAndNotTheDisabledOne(t *testing.T) {
	host := newProviderHost(t, map[string]numenv1connect.AgentServiceHandler{
		disabledProvider:  answeringServer{},
		failingProvider:   &floodAgentServer{},
		answeringProvider: answeringServer{},
	}, disabledProvider)

	_, err := agentbridge.NewRouter(host).GetPrograms(t.Context(), nil)

	if err == nil {
		t.Fatal("the failing provider was not reported")
	}
	if strings.Contains(err.Error(), disabledProvider) {
		t.Errorf("the disabled provider was reported: %v", err)
	}
	if !strings.Contains(err.Error(), "rpc get agent programs") {
		t.Errorf("the failure of the enabled provider was not reported: %v", err)
	}
	for _, id := range host.getLaunched() {
		if id == disabledProvider {
			t.Errorf("the disabled provider was launched")
		}
	}
}

func TestListingWithOnlyDisabledProvidersReportsNothing(t *testing.T) {
	host := newProviderHost(t, map[string]numenv1connect.AgentServiceHandler{
		disabledProvider: answeringServer{},
		failingProvider:  &floodAgentServer{},
	}, disabledProvider, failingProvider)

	programs, err := agentbridge.NewRouter(host).GetPrograms(t.Context(), nil)

	if err != nil {
		t.Errorf("disabled providers were reported: %v", err)
	}
	if len(programs) != 0 {
		t.Errorf("listed %+v from disabled providers", programs)
	}
}

// staleEnabledHost is a provider host that reports every plugin enabled, as a host does for a
// plugin disabled between the check and its activation.
type staleEnabledHost struct {
	*providerHost
}

func (h staleEnabledHost) GetPlugin(id string) (manifest.Discovered, bool) {
	for _, info := range h.GetPluginInfos() {
		if info.ID == id {
			return manifest.Discovered{}, true
		}
	}
	return manifest.Discovered{}, false
}

func TestListingReportsNoProviderDisabledBeforeItsActivation(t *testing.T) {
	host := newProviderHost(t, map[string]numenv1connect.AgentServiceHandler{
		disabledProvider:  answeringServer{},
		failingProvider:   &floodAgentServer{},
		answeringProvider: answeringServer{},
	}, disabledProvider)

	programs, err := agentbridge.NewRouter(staleEnabledHost{host}).GetPrograms(t.Context(), nil)

	if err == nil {
		t.Fatal("the failing provider was not reported")
	}
	if strings.Contains(err.Error(), disabledProvider) {
		t.Errorf("the disabled provider was reported: %v", err)
	}
	if !strings.Contains(err.Error(), "rpc get agent programs") {
		t.Errorf("the failure of the enabled provider was not reported: %v", err)
	}
	if len(programs) != 1 || programs[0].Provider != answeringProvider {
		t.Errorf("listed %+v, want only lark of %s", programs, answeringProvider)
	}
}
