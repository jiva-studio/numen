package container_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// providerServer is an agent plugin that runs the programs it is given.
type providerServer struct {
	numenv1connect.UnimplementedAgentServiceHandler
	programs []*v1.AgentProgram
	onlyWith string
	gate     <-chan struct{}

	mu        sync.Mutex
	requested []*v1.AskAgentRequest
	finished  []string
	commands  []*v1.AgentProgramCommand
}

func (s *providerServer) AskAgent(
	_ context.Context,
	req *connect.Request[v1.AskAgentRequest],
	stream *connect.ServerStream[v1.AskAgentResponse],
) error {
	s.mu.Lock()
	s.requested = append(s.requested, req.Msg)
	s.mu.Unlock()
	if err := stream.Send(&v1.AskAgentResponse{Step: &v1.AskAgentResponse_Text{Text: "answer: " + req.Msg.GetPrompt()}}); err != nil {
		return err
	}
	return stream.Send(&v1.AskAgentResponse{Step: &v1.AskAgentResponse_Stopped{Stopped: "done"}})
}

func (s *providerServer) FinishConversation(
	_ context.Context, req *connect.Request[v1.FinishConversationRequest],
) (*connect.Response[v1.FinishConversationResponse], error) {
	s.mu.Lock()
	s.finished = append(s.finished, req.Msg.GetConversation())
	s.mu.Unlock()
	return connect.NewResponse(&v1.FinishConversationResponse{}), nil
}

func (s *providerServer) getRequested() []*v1.AskAgentRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.requested)
}

func (s *providerServer) getFinished() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.finished)
}

func (s *providerServer) getCommands() []*v1.AgentProgramCommand {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.commands)
}

func (s *providerServer) GetAgentPrograms(
	ctx context.Context, req *connect.Request[v1.GetAgentProgramsRequest],
) (*connect.Response[v1.GetAgentProgramsResponse], error) {
	if s.gate != nil {
		select {
		case <-s.gate:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	s.mu.Lock()
	s.commands = req.Msg.GetCommands()
	s.mu.Unlock()
	if s.onlyWith != "" {
		items := false
		for _, one := range req.Msg.GetCommands() {
			items = items || one.GetProgram() == s.onlyWith
		}
		if !items {
			return connect.NewResponse(&v1.GetAgentProgramsResponse{}), nil
		}
	}
	return connect.NewResponse(&v1.GetAgentProgramsResponse{Programs: s.programs}), nil
}

func hostWith(t *testing.T, capabilities map[string][]string, servers map[string]*providerServer) *plugin.Host {
	t.Helper()

	tmp := t.TempDir()
	for id, declared := range capabilities {
		dir := filepath.Join(tmp, id)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		manifest := fmt.Sprintf(`{
			"id": %q, "name": %q, "version": "1.0.0",
			"capabilities": [%s],
			"targets": ["desktop.editor"],
			"services": ["numen.v1.AgentService"], "backend": {"entrypoint": "bin/x"}
		}`, id, id, `"`+strings.Join(declared, `","`)+`"`)
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	ports := map[string]int{}
	for id, served := range servers {
		mux := http.NewServeMux()
		wiretest.HandlePluginLifecycle(mux)
		mux.Handle(numenv1connect.NewAgentServiceHandler(served))
		server := httptest.NewServer(mux)
		t.Cleanup(server.Close)
		addr, isTCP := server.Listener.Addr().(*net.TCPAddr)
		if !isTCP {
			t.Fatalf("expected *net.TCPAddr, got %T", server.Listener.Addr())
		}
		ports[id] = addr.Port
	}

	host := plugin.NewHost(testsupport.GetMachine(), "desktop.editor", tmp)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}
	host.Processes.SetLauncher(func(ctx context.Context, p manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		serverPort, served := ports[p.Manifest.ID]
		if !served {
			return nil, nil, errors.New("this plugin cannot be started")
		}
		cmd := exec.CommandContext(context.WithoutCancel(ctx), "sleep", "10")
		if err := cmd.Start(); err != nil {
			return nil, nil, err
		}
		t.Cleanup(func() { wiretest.Kill(t, cmd) })
		return cmd.Process, io.NopCloser(strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", serverPort))), nil
	})
	return host
}

const (
	providerOne = "md.test.one"
	providerTwo = "md.test.two"
)

func twoProviders(t *testing.T) (*plugin.Host, *providerServer, *providerServer) {
	t.Helper()
	one := &providerServer{programs: []*v1.AgentProgram{{
		Name: "alpha", Title: "Alpha", IsInstalled: true, IsDefault: true,
		Models: []*v1.AgentModel{{Name: "big", Title: "Big", IsDefault: true}},
	}}}
	two := &providerServer{programs: []*v1.AgentProgram{{Name: "beta", Title: "Beta"}}}
	host := hostWith(t,
		map[string][]string{
			providerOne:      {"ai.agent.provider"},
			providerTwo:      {"ai.agent.provider"},
			"md.test.chatty": {"embedding"},
		},
		map[string]*providerServer{providerOne: one, providerTwo: two})
	return host, one, two
}

func TestAgentProviderWithoutAHostRunsNothing(t *testing.T) {
	assertNoAgentRuns(t, container.NewConfig(container.Config{}).AgentProvider())
}

func TestAgentProviderInSafeModeRunsNothing(t *testing.T) {
	host, _, _ := twoProviders(t)
	assertNoAgentRuns(t, container.NewConfig(container.Config{PluginHost: container.AdaptPluginHost(host), SafeMode: true}).AgentProvider())
}

func assertNoAgentRuns(t *testing.T, provider agentport.AgentProvider) {
	t.Helper()
	programs, err := provider.GetPrograms(t.Context(), nil)
	if err != nil || len(programs) != 0 {
		t.Errorf("listed %v, %v", programs, err)
	}
	if _, err := provider.StartRun(t.Context(), agent.Task{Agent: "alpha"}); err == nil {
		t.Error("a task was taken where no plugin runs")
	}
}

// A host with no plugin that runs agent programs lists none, and takes no task.
func TestAHostWithoutProvidersListsNoPrograms(t *testing.T) {
	host := hostWith(t, map[string][]string{"md.test.chatty": {"embedding"}}, nil)
	provider := (container.NewConfig(container.Config{PluginHost: container.AdaptPluginHost(host)})).AgentProvider()
	if provider == nil {
		t.Fatal("a host has a provider, even where no plugin is one")
	}

	programs, err := provider.GetPrograms(t.Context(), nil)
	if err != nil || len(programs) != 0 {
		t.Errorf("listed %v, %v", programs, err)
	}
	if _, err := provider.StartRun(t.Context(), agent.Task{Agent: "alpha"}); err == nil {
		t.Error("a task was taken where nothing runs the program")
	}
}

// Every plugin with the agent provider capability is queried for its programs,
// in plugin order, and a plugin without it is not queried.
func TestTheProgramsOfEveryProviderAreListedInOrder(t *testing.T) {
	host, one, _ := twoProviders(t)
	provider := (container.NewConfig(container.Config{PluginHost: container.AdaptPluginHost(host)})).AgentProvider()

	programs, err := provider.GetPrograms(t.Context(), map[string][]string{"alpha": {"/bin/alpha", "--x"}})
	if err != nil {
		t.Fatal(err)
	}

	want := []agent.Program{
		{
			Provider: providerOne, Name: "alpha", Title: "Alpha", IsInstalled: true, IsDefault: true,
			Models: []agent.Model{{Name: "big", Title: "Big", IsDefault: true}},
		},
		{Provider: providerTwo, Name: "beta", Title: "Beta", Models: []agent.Model{}},
	}
	if !reflect.DeepEqual(programs, want) {
		t.Errorf("listed %+v, want %+v", programs, want)
	}
	if commands := one.getCommands(); len(commands) != 1 || commands[0].GetProgram() != "alpha" ||
		!slices.Equal(commands[0].GetCommand(), []string{"/bin/alpha", "--x"}) {
		t.Errorf("the provider received %v", commands)
	}
}

// A provider that cannot be started is reported alongside the programs of the
// others.
func TestAProviderThatFailsIsReportedAdjacentTheOthers(t *testing.T) {
	served := &providerServer{programs: []*v1.AgentProgram{{Name: "alpha", Title: "Alpha"}}}
	host := hostWith(t,
		map[string][]string{providerOne: {"ai.agent.provider"}, providerTwo: {"ai.agent.provider"}},
		map[string]*providerServer{providerTwo: served})
	provider := (container.NewConfig(container.Config{PluginHost: container.AdaptPluginHost(host)})).AgentProvider()

	programs, err := provider.GetPrograms(t.Context(), nil)
	if err == nil || !strings.Contains(err.Error(), providerOne) {
		t.Errorf("the failure is %v", err)
	}
	if len(programs) != 1 || programs[0].Name != "alpha" {
		t.Errorf("listed %+v", programs)
	}
}

// A task goes to the plugin that runs the program it names, with everything it was given.
func TestATaskGoesToThePluginThatRunsItsProgram(t *testing.T) {
	host, one, two := twoProviders(t)
	provider := (container.NewConfig(container.Config{PluginHost: container.AdaptPluginHost(host)})).AgentProvider()

	run, err := provider.StartRun(t.Context(), agent.Task{
		Question: "hello", Agent: "beta", Model: "m", MaxSteps: 3, Root: "/vault",
		Endpoint: agent.ToolEndpoint{URL: "http://127.0.0.1:1/mcp", Token: "t"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var steps []agent.Step
	for step := range run.GetSteps() {
		steps = append(steps, step)
	}

	if len(one.getRequested()) != 0 || len(two.getRequested()) != 1 {
		t.Fatalf("the first plugin was asked %d times and the second %d", len(one.getRequested()), len(two.getRequested()))
	}
	if got := two.getRequested()[0]; got.GetPrompt() != "hello" || got.GetAgent() != "beta" || got.GetModel() != "m" ||
		got.GetMaxSteps() != 3 || got.GetRoot() != "/vault" || got.GetToolsUrl() != "http://127.0.0.1:1/mcp" ||
		got.GetToolsToken() != "t" {
		t.Errorf("the plugin was asked %v", got)
	}
	if len(steps) != 2 || steps[0].Kind != domain.StepSay || steps[0].Text != "answer: hello" ||
		steps[1].Kind != domain.StepStop {
		t.Errorf("the steps are %+v", steps)
	}
}

func TestATaskNamingNoRunnableProgramIsRefused(t *testing.T) {
	host, one, two := twoProviders(t)
	provider := (container.NewConfig(container.Config{PluginHost: container.AdaptPluginHost(host)})).AgentProvider()

	for _, name := range []string{"", "gamma"} {
		if _, err := provider.StartRun(t.Context(), agent.Task{Question: "q", Agent: name}); err == nil {
			t.Errorf("a task naming %q was taken", name)
		}
	}
	if len(one.getRequested())+len(two.getRequested()) != 0 {
		t.Error("a plugin was asked for a task nobody runs")
	}
}

// Ending a conversation is sent to the plugin that started it, and ending a
// conversation that was never started sends nothing.
func TestEndConversation_RoutesToStartingPlugin(t *testing.T) {
	host, one, two := twoProviders(t)
	provider := (container.NewConfig(container.Config{PluginHost: container.AdaptPluginHost(host)})).AgentProvider()

	run, err := provider.StartRun(t.Context(), agent.Task{Question: "q", Agent: "alpha", Conversation: "c-1"})
	if err != nil {
		t.Fatal(err)
	}
	for range run.GetSteps() {
	}

	if err := provider.EndConversation(t.Context(), "c-1"); err != nil {
		t.Fatal(err)
	}
	if err := provider.EndConversation(t.Context(), "c-1"); err != nil {
		t.Fatal(err)
	}
	if err := provider.EndConversation(t.Context(), "never"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(one.getFinished(), []string{"c-1"}) || len(two.getFinished()) != 0 {
		t.Errorf("the first plugin finished %v and the second %v", one.getFinished(), two.getFinished())
	}
}

// The window is offered the programs the plugins run, and the models of the installed ones.
func TestAgentContainer_ExposesProgramsAndModels(t *testing.T) {
	host, _, _ := twoProviders(t)
	settings := filepath.Join(t.TempDir(), "numen.json")
	if err := os.WriteFile(settings, []byte(`{"app":{"agent":{"alpha":{"model":"mine"}}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	models := (container.NewConfig(container.Config{
		PluginHost: container.AdaptPluginHost(host),
		Paths:      container.Paths{SettingsPath: settings},
	})).Models()()

	byPath := map[string][]settingsdomain.Model{}
	for _, one := range models {
		key := strings.Join(one.Path, ".")
		byPath[key] = append(byPath[key], one)
	}
	uses := byPath["app.agent.use"]
	if len(uses) != 3 || uses[0].Name != "alpha" || uses[1].Name != "beta" || uses[2].Name != "" {
		t.Fatalf("the programs are %+v", uses)
	}
	alpha := byPath["app.agent.alpha.model"]
	if len(alpha) != 2 || alpha[0].Name != "big" || alpha[1].Name != "mine" {
		t.Errorf("the models of alpha are %+v", alpha)
	}
	if len(byPath["app.agent.beta.model"]) != 0 {
		t.Errorf("a program that is not installed offers %+v", byPath["app.agent.beta.model"])
	}
}

// A conversation started on two plugins is ended on both.
func TestEndConversation_EndsOnEveryPluginUsed(t *testing.T) {
	host, one, two := twoProviders(t)
	provider := (container.NewConfig(container.Config{PluginHost: container.AdaptPluginHost(host)})).AgentProvider()

	for _, program := range []string{"alpha", "beta", "alpha"} {
		run, err := provider.StartRun(t.Context(), agent.Task{Question: "q", Agent: program, Conversation: "c-1"})
		if err != nil {
			t.Fatal(err)
		}
		for range run.GetSteps() {
		}
	}

	if err := provider.EndConversation(t.Context(), "c-1"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(one.getFinished(), []string{"c-1"}) || !slices.Equal(two.getFinished(), []string{"c-1"}) {
		t.Errorf("the first plugin finished %v and the second %v", one.getFinished(), two.getFinished())
	}
}

// A plugin that cannot be queried for its programs is reported to the error
// handler, and the programs of the others are returned.
func TestModels_ReportsProviderListFailureToErrorHandler(t *testing.T) {
	served := &providerServer{programs: []*v1.AgentProgram{{Name: "alpha", Title: "Alpha", IsInstalled: true}}}
	host := hostWith(t,
		map[string][]string{providerOne: {"ai.agent.provider"}, providerTwo: {"ai.agent.provider"}},
		map[string]*providerServer{providerTwo: served})

	settingsFile := filepath.Join(t.TempDir(), "numen.json")
	if err := os.WriteFile(settingsFile, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var notified []error
	cfg := container.NewConfig(container.Config{
		PluginHost:   container.AdaptPluginHost(host),
		ErrorHandler: func(err error) { notified = append(notified, err) },
		Paths:        container.Paths{SettingsPath: settingsFile},
	})
	models := cfg.Models()()

	if len(notified) != 1 || !strings.Contains(notified[0].Error(), providerOne) {
		t.Errorf("the error handler received %v", notified)
	}
	var names []string
	for _, one := range models {
		if strings.Join(one.Path, ".") == "app.agent.use" {
			names = append(names, one.Name)
		}
	}
	if !slices.Equal(names, []string{"alpha", ""}) {
		t.Errorf("the programs offered are %q", names)
	}
}

// A program the plugin lists only under the command line the installation set is found, and run,
// under that command line.
func TestAProgramAvailableOnlyThroughItsCommandIsFound(t *testing.T) {
	served := &providerServer{
		programs: []*v1.AgentProgram{{Name: "gamma", Title: "Gamma", IsInstalled: true}},
		onlyWith: "gamma",
	}
	host := hostWith(t, map[string][]string{providerOne: {"ai.agent.provider"}},
		map[string]*providerServer{providerOne: served})
	provider := (container.NewConfig(container.Config{PluginHost: container.AdaptPluginHost(host)})).AgentProvider()

	if _, err := provider.StartRun(t.Context(), agent.Task{Question: "q", Agent: "gamma"}); err == nil ||
		!strings.Contains(err.Error(), "no agent plugin runs") {
		t.Fatalf("a program nothing lists was run: %v", err)
	}
	if len(served.getRequested()) != 0 {
		t.Fatal("the plugin was asked for a program it did not list")
	}
	run, err := provider.StartRun(t.Context(), agent.Task{Question: "q", Agent: "gamma", Command: []string{"/opt/gamma"}})
	if err != nil {
		t.Fatal(err)
	}
	for range run.GetSteps() {
	}
	if request := served.getRequested(); len(request) != 1 || !slices.Equal(request[0].GetCommand(), []string{"/opt/gamma"}) {
		t.Errorf("the plugin was asked %v", request)
	}
}
