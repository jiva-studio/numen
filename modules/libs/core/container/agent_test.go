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

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// providerServer is an agent plugin that runs the programs it is given.
type providerServer struct {
	numenv1connect.UnimplementedAgentServiceHandler
	programs []*v1.AgentProgram
	// onlyWith, when set, is the program the plugin lists only under a command line set for it.
	onlyWith string

	mu       sync.Mutex
	asked    []*v1.AskAgentRequest
	finished []string
	commands []*v1.AgentProgramCommand
}

func (s *providerServer) AskAgent(
	_ context.Context,
	req *connect.Request[v1.AskAgentRequest],
	stream *connect.ServerStream[v1.AskAgentResponse],
) error {
	s.mu.Lock()
	s.asked = append(s.asked, req.Msg)
	s.mu.Unlock()
	_ = stream.Send(&v1.AskAgentResponse{Step: &v1.AskAgentResponse_Said{Said: "answer: " + req.Msg.GetPrompt()}})
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

// getAsked are the tasks the plugin was given.
func (s *providerServer) getAsked() []*v1.AskAgentRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.asked)
}

// getFinished are the conversations the plugin was told are over.
func (s *providerServer) getFinished() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.finished)
}

// getCommands are the command lines the plugin was told about.
func (s *providerServer) getCommands() []*v1.AgentProgramCommand {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.commands)
}

func (s *providerServer) GetAgentPrograms(
	_ context.Context, req *connect.Request[v1.GetAgentProgramsRequest],
) (*connect.Response[v1.GetAgentProgramsResponse], error) {
	s.mu.Lock()
	s.commands = req.Msg.GetCommands()
	s.mu.Unlock()
	if s.onlyWith != "" {
		listed := false
		for _, one := range req.Msg.GetCommands() {
			listed = listed || one.GetProgram() == s.onlyWith
		}
		if !listed {
			return connect.NewResponse(&v1.GetAgentProgramsResponse{}), nil
		}
	}
	return connect.NewResponse(&v1.GetAgentProgramsResponse{Programs: s.programs}), nil
}

// hostWith is a plugin host over plugins that declare the capabilities given, each served by the
// server with its identifier. A plugin with no server cannot be started.
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
			"backend": {"type": "process", "entrypoint": "bin/x", "serviceName": "numen.v1.AgentService"}
		}`, id, id, `"`+strings.Join(declared, `","`)+`"`)
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	ports := map[string]int{}
	for id, served := range servers {
		mux := http.NewServeMux()
		mux.Handle(numenv1connect.NewAgentServiceHandler(served))
		server := httptest.NewServer(mux)
		t.Cleanup(server.Close)
		ports[id] = server.Listener.Addr().(*net.TCPAddr).Port
	}

	host := plugin.NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatal(err)
	}
	host.SetLauncher(func(_ context.Context, p plugin.DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		serverPort, served := ports[p.Manifest.ID]
		if !served {
			return nil, nil, errors.New("this plugin cannot be started")
		}
		cmd := exec.Command("sleep", "10")
		if err := cmd.Start(); err != nil {
			return nil, nil, err
		}
		t.Cleanup(func() {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		})
		return cmd.Process, io.NopCloser(strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", serverPort))), nil
	})
	return host
}

const (
	providerOne = "md.test.one"
	providerTwo = "md.test.two"
)

// twoProviders is a host over two plugins that run agent programs, and a third that is no provider.
func twoProviders(t *testing.T) (*plugin.Host, *providerServer, *providerServer) {
	t.Helper()
	one := &providerServer{programs: []*v1.AgentProgram{{
		Name: "alpha", Title: "Alpha", IsInstalled: true, IsDefault: true,
		Models: []*v1.AgentModel{{Name: "big", Title: "Big", IsDefault: true}},
	}}}
	two := &providerServer{programs: []*v1.AgentProgram{{Name: "beta", Title: "Beta"}}}
	host := hostWith(t,
		map[string][]string{
			providerOne:      {"agent", "ai.agent.provider"},
			providerTwo:      {"ai.agent.provider"},
			"md.test.chatty": {"agent"},
		},
		map[string]*providerServer{providerOne: one, providerTwo: two})
	return host, one, two
}

func TestNoAgentProviderWithoutAHost(t *testing.T) {
	if provider := (container.Config{}).AgentProvider(); provider != nil {
		t.Fatalf("expected no provider, got %v", provider)
	}
}

func TestNoAgentProviderInSafeMode(t *testing.T) {
	host, _, _ := twoProviders(t)
	if provider := (container.Config{PluginHost: host, SafeMode: true}).AgentProvider(); provider != nil {
		t.Fatalf("expected no provider, got %v", provider)
	}
}

// A host with no plugin that runs agent programs lists none, and takes no task.
func TestAHostWithoutProvidersListsNoPrograms(t *testing.T) {
	host := hostWith(t, map[string][]string{"md.test.chatty": {"agent"}}, nil)
	provider := (container.Config{PluginHost: host}).AgentProvider()
	if provider == nil {
		t.Fatal("a host has a provider, even where no plugin is one")
	}

	programs, err := provider.GetPrograms(t.Context(), nil)
	if err != nil || len(programs) != 0 {
		t.Errorf("listed %v, %v", programs, err)
	}
	if _, err := provider.StartRun(t.Context(), port.Task{Agent: "alpha"}); err == nil {
		t.Error("a task was taken where nothing runs the program")
	}
}

// Every plugin with the agent provider capability is asked for its programs, in the order of the
// plugins, and a plugin without it is not asked.
func TestTheProgramsOfEveryProviderAreListedInOrder(t *testing.T) {
	host, one, _ := twoProviders(t)
	provider := (container.Config{PluginHost: host}).AgentProvider()

	programs, err := provider.GetPrograms(t.Context(), map[string][]string{"alpha": {"/bin/alpha", "--x"}})
	if err != nil {
		t.Fatal(err)
	}

	want := []port.AgentProgram{
		{
			Provider: providerOne, Name: "alpha", Title: "Alpha", IsInstalled: true, IsDefault: true,
			Models: []port.AgentModel{{Name: "big", Title: "Big", IsDefault: true}},
		},
		{Provider: providerTwo, Name: "beta", Title: "Beta", Models: []port.AgentModel{}},
	}
	if !reflect.DeepEqual(programs, want) {
		t.Errorf("listed %+v, want %+v", programs, want)
	}
	if commands := one.getCommands(); len(commands) != 1 || commands[0].GetProgram() != "alpha" ||
		!slices.Equal(commands[0].GetCommand(), []string{"/bin/alpha", "--x"}) {
		t.Errorf("the provider was told %v", commands)
	}
}

// A provider that cannot be started is told beside the programs of the others.
func TestAProviderThatFailsIsToldBesideTheOthers(t *testing.T) {
	served := &providerServer{programs: []*v1.AgentProgram{{Name: "alpha", Title: "Alpha"}}}
	host := hostWith(t,
		map[string][]string{providerOne: {"ai.agent.provider"}, providerTwo: {"ai.agent.provider"}},
		map[string]*providerServer{providerTwo: served})
	provider := (container.Config{PluginHost: host}).AgentProvider()

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
	provider := (container.Config{PluginHost: host}).AgentProvider()

	run, err := provider.StartRun(t.Context(), port.Task{
		Question: "hello", Agent: "beta", Model: "m", MaxSteps: 3, Root: "/vault",
		Endpoint: port.ToolEndpoint{URL: "http://127.0.0.1:1/mcp", Token: "t"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var steps []port.Step
	for step := range run.GetSteps() {
		steps = append(steps, step)
	}

	if len(one.getAsked()) != 0 || len(two.getAsked()) != 1 {
		t.Fatalf("the first plugin was asked %d times and the second %d", len(one.getAsked()), len(two.getAsked()))
	}
	if got := two.getAsked()[0]; got.GetPrompt() != "hello" || got.GetAgent() != "beta" || got.GetModel() != "m" ||
		got.GetMaxSteps() != 3 || got.GetRoot() != "/vault" || got.GetToolsUrl() != "http://127.0.0.1:1/mcp" ||
		got.GetToolsToken() != "t" {
		t.Errorf("the plugin was asked %v", got)
	}
	if len(steps) != 2 || steps[0].Kind != domain.StepSay || steps[0].Text != "answer: hello" ||
		steps[1].Kind != domain.StepStop {
		t.Errorf("the steps are %+v", steps)
	}
}

// A task naming no program, or one nobody runs, is refused.
func TestATaskNamingNoRunnableProgramIsRefused(t *testing.T) {
	host, one, two := twoProviders(t)
	provider := (container.Config{PluginHost: host}).AgentProvider()

	for _, name := range []string{"", "gamma"} {
		if _, err := provider.StartRun(t.Context(), port.Task{Question: "q", Agent: name}); err == nil {
			t.Errorf("a task naming %q was taken", name)
		}
	}
	if len(one.getAsked())+len(two.getAsked()) != 0 {
		t.Error("a plugin was asked for a task nobody runs")
	}
}

// Ending a conversation goes to the plugin it was asked in, and ending one that was asked nowhere
// goes nowhere.
func TestAConversationEndsWhereItWasAsked(t *testing.T) {
	host, one, two := twoProviders(t)
	provider := (container.Config{PluginHost: host}).AgentProvider()

	run, err := provider.StartRun(t.Context(), port.Task{Question: "q", Agent: "alpha", Conversation: "c-1"})
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
func TestTheWindowIsOfferedWhatThePluginsList(t *testing.T) {
	host, _, _ := twoProviders(t)
	settings := filepath.Join(t.TempDir(), "numen.json")
	if err := os.WriteFile(settings, []byte(`{"agent":{"alpha":{"model":"mine"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	models := (container.Config{PluginHost: host, SettingsPath: settings}).Models()()

	byPath := map[string][]port.Model{}
	for _, one := range models {
		key := strings.Join(one.Path, ".")
		byPath[key] = append(byPath[key], one)
	}
	uses := byPath["agent.use"]
	if len(uses) != 3 || uses[0].Name != "alpha" || uses[1].Name != "beta" || uses[2].Name != "" {
		t.Fatalf("the programs are %+v", uses)
	}
	alpha := byPath["agent.alpha.model"]
	if len(alpha) != 2 || alpha[0].Name != "big" || alpha[1].Name != "mine" {
		t.Errorf("the models of alpha are %+v", alpha)
	}
	if len(byPath["agent.beta.model"]) != 0 {
		t.Errorf("a program that is not installed offers %+v", byPath["agent.beta.model"])
	}
}

// A conversation asked on two plugins is ended on both.
func TestAConversationAskedOnTwoPluginsEndsOnBoth(t *testing.T) {
	host, one, two := twoProviders(t)
	provider := (container.Config{PluginHost: host}).AgentProvider()

	for _, program := range []string{"alpha", "beta", "alpha"} {
		run, err := provider.StartRun(t.Context(), port.Task{Question: "q", Agent: program, Conversation: "c-1"})
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

// A plugin that cannot be asked for its programs is told to whoever handles what went wrong, and
// the programs of the others are offered.
func TestAPluginThatCannotListItsProgramsIsToldToTheErrorHandler(t *testing.T) {
	served := &providerServer{programs: []*v1.AgentProgram{{Name: "alpha", Title: "Alpha", IsInstalled: true}}}
	host := hostWith(t,
		map[string][]string{providerOne: {"ai.agent.provider"}, providerTwo: {"ai.agent.provider"}},
		map[string]*providerServer{providerTwo: served})

	settingsFile := filepath.Join(t.TempDir(), "numen.json")
	if err := os.WriteFile(settingsFile, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var told []error
	cfg := container.Config{
		PluginHost: host, SettingsPath: settingsFile,
		ErrorHandler: func(err error) { told = append(told, err) },
	}
	models := cfg.Models()()

	if len(told) != 1 || !strings.Contains(told[0].Error(), providerOne) {
		t.Errorf("the error handler was told %v", told)
	}
	var names []string
	for _, one := range models {
		if strings.Join(one.Path, ".") == "agent.use" {
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
	provider := (container.Config{PluginHost: host}).AgentProvider()

	if _, err := provider.StartRun(t.Context(), port.Task{Question: "q", Agent: "gamma"}); err == nil ||
		!strings.Contains(err.Error(), "no agent plugin runs") {
		t.Fatalf("a program nothing lists was run: %v", err)
	}
	if len(served.getAsked()) != 0 {
		t.Fatal("the plugin was asked for a program it did not list")
	}
	run, err := provider.StartRun(t.Context(), port.Task{Question: "q", Agent: "gamma", Command: []string{"/opt/gamma"}})
	if err != nil {
		t.Fatal(err)
	}
	for range run.GetSteps() {
	}
	if asked := served.getAsked(); len(asked) != 1 || !slices.Equal(asked[0].GetCommand(), []string{"/opt/gamma"}) {
		t.Errorf("the plugin was asked %v", asked)
	}
}
