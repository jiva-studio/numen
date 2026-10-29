package container_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/manifestfile"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/port/agentport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

const (
	agentPlugin = "md.test.agent"
	vaultPlugin = "md.test.vault"
	cardPlugin  = "md.test.cards"
	cardKey     = cardPlugin + "/card"
)

// toolCall is one call the agent stand-in makes.
type toolCall struct {
	name      string
	arguments string
}

// toolAgent is an agent plugin that connects to the tools of its task over MCP: it
// lists them, makes the calls it is given and records each result.
type toolAgent struct {
	numenv1connect.UnimplementedAgentServiceHandler
	calls []toolCall

	mu       sync.Mutex
	requests []*v1.AskAgentRequest
	names    []string
	results  []string
}

func (a *toolAgent) GetAgentPrograms(
	context.Context, *connect.Request[v1.GetAgentProgramsRequest],
) (*connect.Response[v1.GetAgentProgramsResponse], error) {
	return connect.NewResponse(&v1.GetAgentProgramsResponse{Programs: []*v1.AgentProgram{
		{Name: "alpha", Title: "Alpha", IsInstalled: true, IsDefault: true},
	}}), nil
}

func (a *toolAgent) AskAgent(
	ctx context.Context, req *connect.Request[v1.AskAgentRequest], stream *connect.ServerStream[v1.AskAgentResponse],
) error {
	a.mu.Lock()
	a.requests = append(a.requests, req.Msg)
	a.mu.Unlock()
	if req.Msg.GetToolsUrl() != "" {
		if err := a.callTools(ctx, req.Msg.GetToolsUrl(), req.Msg.GetToolsToken()); err != nil {
			return stream.Send(&v1.AskAgentResponse{Step: &v1.AskAgentResponse_Stopped{Stopped: err.Error()}})
		}
	}
	return stream.Send(&v1.AskAgentResponse{Step: &v1.AskAgentResponse_Stopped{Stopped: "done"}})
}

// callTools lists the tools at the url and makes the calls of the stand-in.
func (a *toolAgent) callTools(ctx context.Context, url, token string) error {
	session, err := connectTools(ctx, url, token)
	if err != nil {
		return err
	}
	defer func() { _ = session.Close() }()

	var names []string
	for tool, err := range session.Tools(ctx, nil) {
		if err != nil {
			return err
		}
		names = append(names, tool.Name)
	}
	var results []string
	for _, call := range a.calls {
		result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: call.name, Arguments: json.RawMessage(call.arguments)})
		if err != nil {
			return err
		}
		for _, content := range result.Content {
			if text, isText := content.(*sdk.TextContent); isText {
				results = append(results, text.Text)
			}
		}
	}
	a.mu.Lock()
	a.names, a.results = names, results
	a.mu.Unlock()
	return nil
}

func (a *toolAgent) getRequests() []*v1.AskAgentRequest {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.requests)
}

func (a *toolAgent) getToolNames() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.names)
}

func (a *toolAgent) getResults() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.results)
}

// bearerTransport presents the token on every request.
type bearerTransport struct{ token string }

func (b bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(req)
}

// connectTools opens an MCP session on the tools at the url, presenting the token.
func connectTools(ctx context.Context, url, token string) (*sdk.ClientSession, error) {
	client := sdk.NewClient(&sdk.Implementation{Name: "stand-in", Version: "1"}, nil)
	return client.Connect(ctx, &sdk.StreamableClientTransport{
		Endpoint:             url,
		HTTPClient:           &http.Client{Transport: bearerTransport{token: token}},
		DisableStandaloneSSE: true,
	}, nil)
}

// toolPlugin is a plugin that lists tools and answers every call with its name and arguments.
type toolPlugin struct {
	numenv1connect.UnimplementedPluginToolServiceHandler
	tools []*v1.PluginTool

	mu      sync.Mutex
	lists   int
	request []*v1.CallToolRequest
}

func (p *toolPlugin) ListTools(
	context.Context, *connect.Request[v1.ListToolsRequest],
) (*connect.Response[v1.ListToolsResponse], error) {
	p.mu.Lock()
	p.lists++
	p.mu.Unlock()
	return connect.NewResponse(&v1.ListToolsResponse{Tools: p.tools}), nil
}

func (p *toolPlugin) CallTool(
	_ context.Context, req *connect.Request[v1.CallToolRequest],
) (*connect.Response[v1.CallToolResponse], error) {
	p.mu.Lock()
	p.request = append(p.request, req.Msg)
	p.mu.Unlock()
	text := fmt.Sprintf("%s %s %s", req.Msg.GetName(), req.Msg.GetArguments(), req.Msg.GetContext()[cardKey])
	return connect.NewResponse(&v1.CallToolResponse{Result: &v1.ToolResult{Text: text}}), nil
}

func (p *toolPlugin) getRequests() []*v1.CallToolRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.request)
}

func (p *toolPlugin) countLists() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.lists
}

func newPluginTool(name string, effect v1.ToolEffect, servers ...v1.ToolServer) *v1.PluginTool {
	return &v1.PluginTool{
		Name: name, Title: name, Effect: effect, Servers: servers,
		InputSchema: []byte(`{"type":"object","properties":{"path":{"type":"string"}}}`),
	}
}

// testPlugin is one plugin of a test host: its manifest and the services its backend serves.
type testPlugin struct {
	id           string
	capabilities []string
	permissions  []string
	mount        func(mux *http.ServeMux)
}

// hostOver is a plugin host over the plugins, each served in this process.
func hostOver(t *testing.T, plugins ...testPlugin) *plugin.Host {
	t.Helper()
	tmp := t.TempDir()
	ports := map[string]int{}
	for _, one := range plugins {
		dir := filepath.Join(tmp, one.id)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		capabilities, _ := json.Marshal(one.capabilities)
		permissions, _ := json.Marshal(one.permissions)
		manifest := fmt.Sprintf(`{
			"id": %q, "name": %q, "version": "1.0.0",
			"capabilities": %s, "permissions": %s,
			"targets": ["desktop.editor"],
			"backend": {"type": "process", "entrypoint": "bin/x", "serviceName": "numen.v1.PluginToolService"}
		}`, one.id, one.id, capabilities, permissions)
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o600); err != nil {
			t.Fatal(err)
		}
		mux := http.NewServeMux()
		testsupport.HandlePluginLifecycle(mux)
		one.mount(mux)
		server := httptest.NewServer(mux)
		t.Cleanup(server.Close)
		ports[one.id] = server.Listener.Addr().(*net.TCPAddr).Port
	}

	host := plugin.NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatal(err)
	}
	host.SetLauncher(func(_ context.Context, p manifestfile.DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		port, isServed := ports[p.Manifest.ID]
		if !isServed {
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
		return cmd.Process, io.NopCloser(strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", port))), nil
	})
	return host
}

func agentAt(agent *toolAgent) testPlugin {
	return testPlugin{
		id: agentPlugin, capabilities: []string{"ai.agent.provider"},
		mount: func(mux *http.ServeMux) { mux.Handle(numenv1connect.NewAgentServiceHandler(agent)) },
	}
}

func toolsAt(id string, permissions []string, tools *toolPlugin) testPlugin {
	return testPlugin{
		id: id, capabilities: []string{"tools"}, permissions: permissions,
		mount: func(mux *http.ServeMux) { mux.Handle(numenv1connect.NewPluginToolServiceHandler(tools)) },
	}
}

// vaultAndCards is a host with an agent, a plugin of vault tools and a plugin of card tools.
func vaultAndCards(t *testing.T, agent *toolAgent) (*plugin.Host, *toolPlugin, *toolPlugin) {
	t.Helper()
	vault := &toolPlugin{tools: []*v1.PluginTool{
		newPluginTool("note_read", v1.ToolEffect_TOOL_EFFECT_READ_ONLY, v1.ToolServer_TOOL_SERVER_FULL, v1.ToolServer_TOOL_SERVER_READING),
		newPluginTool("note_write", v1.ToolEffect_TOOL_EFFECT_READ_WRITE),
	}}
	cards := &toolPlugin{tools: []*v1.PluginTool{
		newPluginTool("card_current", v1.ToolEffect_TOOL_EFFECT_READ_ONLY, v1.ToolServer_TOOL_SERVER_FULL),
	}}
	host := hostOver(t,
		agentAt(agent),
		toolsAt(vaultPlugin, []string{"vault:read", "vault:write"}, vault),
		toolsAt(cardPlugin, []string{"vault:read", "vault:write"}, cards),
	)
	return host, vault, cards
}

// runToEnd starts the task and reads its steps to the end, then stops the run.
func runToEnd(t *testing.T, agent agentport.Agent, task agentport.Task) []agentport.Step {
	t.Helper()
	run, err := agent.StartRun(t.Context(), task)
	if err != nil {
		t.Fatal(err)
	}
	var steps []agentport.Step
	for step := range run.GetSteps() {
		steps = append(steps, step)
	}
	if err := run.Stop(); err != nil {
		t.Fatal(err)
	}
	return steps
}

// An agent run in the editor receives its own endpoint: the agent lists the tools
// of every tools plugin there, and a call is routed to the plugin that listed the tool,
// with the run's context.
func TestAnEditorRunCallsTheToolsOfThePluginsWithItsContext(t *testing.T) {
	agent := &toolAgent{calls: []toolCall{
		{name: "note_read", arguments: `{"path":"Garden.md"}`},
		{name: "card_current", arguments: `{}`},
	}}
	host, vault, cards := vaultAndCards(t, agent)
	assembly := container.NewEditorAssembly(container.Config{PluginHost: host}, nil, nil)

	requestContext := map[string]string{cardKey: "c-7"}
	steps := runToEnd(t, assembly.GetAgent(), agentport.Task{Question: "q", Agent: "alpha", Context: requestContext})

	if last := steps[len(steps)-1]; last.Kind != domain.StepStop || last.Detail != "done" {
		t.Fatalf("the run stopped with %+v", last)
	}
	request := agent.getRequests()[0]
	if !strings.HasPrefix(request.GetToolsUrl(), "http://127.0.0.1:") || request.GetToolsToken() == "" {
		t.Errorf("the agent was given the tools at %q with token %q", request.GetToolsUrl(), request.GetToolsToken())
	}
	if want := []string{"card_current", "note_read", "note_write"}; !slices.Equal(request.GetTools(), want) {
		t.Errorf("the agent may call %v, want %v", request.GetTools(), want)
	}
	if kinds := getDeclaredKinds(request); !maps.Equal(kinds, map[string]string{"note_read": "read", "note_write": "edit", "card_current": "read"}) {
		t.Errorf("the tools are declared as %v", kinds)
	}
	if names := agent.getToolNames(); !slices.Equal(names, []string{"card_current", "note_read", "note_write"}) {
		t.Errorf("the endpoint lists %v", names)
	}
	if results := agent.getResults(); !slices.Equal(results, []string{`note_read {"path":"Garden.md"} c-7`, "card_current {} c-7"}) {
		t.Errorf("the calls returned %v", results)
	}
	for _, served := range []*toolPlugin{vault, cards} {
		if requests := served.getRequests(); len(requests) != 1 || !maps.Equal(requests[0].GetContext(), requestContext) {
			t.Errorf("the plugin was called with %v", requests)
		}
	}
}

func getDeclaredKinds(request *v1.AskAgentRequest) map[string]string {
	kinds := map[string]string{}
	for _, one := range request.GetToolDeclarations() {
		kinds[one.GetName()] = one.GetKind()
	}
	return kinds
}

// The window of the flashcards serves its runs the tools too.
func TestASettingsAgentRunCallsTheToolsOfThePlugins(t *testing.T) {
	agent := &toolAgent{calls: []toolCall{{name: "card_current", arguments: `{}`}}}
	host, _, cards := vaultAndCards(t, agent)
	settingsAgent := openSettingsAgent(t, host, `{"agent": {"use": "alpha"}}`)

	runToEnd(t, settingsAgent, agentport.Task{Question: "q", Context: map[string]string{cardKey: "c-9"}})

	if results := agent.getResults(); !slices.Equal(results, []string{"card_current {} c-9"}) {
		t.Errorf("the calls returned %v", results)
	}
	if requests := cards.getRequests(); len(requests) != 1 || requests[0].GetContext()[cardKey] != "c-9" {
		t.Errorf("the plugin was called with %v", requests)
	}
}

// A tool is offered only when its plugin holds the permission for its effect,
// and only on the servers it names; a plugin that does not declare the tools
// capability is not queried.
func TestOnlyTheToolsTheWindowAllowsAreServed(t *testing.T) {
	agent := &toolAgent{}
	reader := &toolPlugin{tools: []*v1.PluginTool{
		newPluginTool("note_read", v1.ToolEffect_TOOL_EFFECT_READ_ONLY),
		newPluginTool("note_write", v1.ToolEffect_TOOL_EFFECT_READ_WRITE),
		newPluginTool("note_quote", v1.ToolEffect_TOOL_EFFECT_READ_ONLY, v1.ToolServer_TOOL_SERVER_READING),
		newPluginTool("note_guess", v1.ToolEffect_TOOL_EFFECT_UNSPECIFIED),
	}}
	undeclared := &toolPlugin{tools: []*v1.PluginTool{newPluginTool("card_current", v1.ToolEffect_TOOL_EFFECT_READ_ONLY)}}
	host := hostOver(t,
		agentAt(agent),
		toolsAt(vaultPlugin, []string{"vault:read"}, reader),
		testPlugin{
			id: cardPlugin, permissions: []string{"vault:read"},
			mount: func(mux *http.ServeMux) { mux.Handle(numenv1connect.NewPluginToolServiceHandler(undeclared)) },
		},
	)
	assembly := container.NewEditorAssembly(container.Config{PluginHost: host}, nil, nil)

	runToEnd(t, assembly.GetAgent(), agentport.Task{Question: "q", Agent: "alpha"})

	if names := agent.getToolNames(); !slices.Equal(names, []string{"note_read"}) {
		t.Errorf("the endpoint lists %v", names)
	}
	if undeclared.countLists() != 0 {
		t.Error("a plugin that does not declare the tools capability was asked for its tools")
	}
}

// A run with no tool to call is given no endpoint.
func TestARunWithNoToolPluginIsGivenNoEndpoint(t *testing.T) {
	agent := &toolAgent{}
	host := hostOver(t, agentAt(agent))
	assembly := container.NewEditorAssembly(container.Config{PluginHost: host}, nil, nil)

	runToEnd(t, assembly.GetAgent(), agentport.Task{Question: "q", Agent: "alpha"})

	if request := agent.getRequests()[0]; request.GetToolsUrl() != "" || len(request.GetTools()) != 0 {
		t.Errorf("the agent was given the tools at %q: %v", request.GetToolsUrl(), request.GetTools())
	}
}

// The endpoint is gone once the run is stopped.
func TestTheEndpointClosesWhenTheRunStops(t *testing.T) {
	agent := &toolAgent{}
	host, _, _ := vaultAndCards(t, agent)
	assembly := container.NewEditorAssembly(container.Config{PluginHost: host}, nil, nil)

	runToEnd(t, assembly.GetAgent(), agentport.Task{Question: "q", Agent: "alpha"})

	request := agent.getRequests()[0]
	if request.GetToolsUrl() == "" {
		t.Fatal("the run was given no endpoint")
	}
	if _, err := connectTools(t.Context(), request.GetToolsUrl(), request.GetToolsToken()); err == nil {
		t.Error("the endpoint still answers after the run stopped")
	}
}

// While the run is active, the endpoint accepts its token and rejects any other.
func TestToolsEndpoint_RejectsWrongToken(t *testing.T) {
	agent := &toolAgent{}
	host, _, _ := vaultAndCards(t, agent)
	run, err := container.NewEditorAssembly(container.Config{PluginHost: host}, nil, nil).GetAgent().
		StartRun(t.Context(), agentport.Task{Question: "q", Agent: "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = run.Stop() })
	for range run.GetSteps() {
	}

	request := agent.getRequests()[0]
	if _, err := connectTools(t.Context(), request.GetToolsUrl(), "another"); err == nil {
		t.Error("the endpoint answered another token")
	}
	session, err := connectTools(t.Context(), request.GetToolsUrl(), request.GetToolsToken())
	if err != nil {
		t.Fatalf("the endpoint refuses the token of its run: %v", err)
	}
	if err := session.Close(); err != nil {
		t.Error(err)
	}
}

// Every call of a run is made in the vault the run works, which the card tools read.
func TestACardToolCallCarriesTheVaultOfTheRun(t *testing.T) {
	agent := &toolAgent{calls: []toolCall{{name: "card_current", arguments: `{}`}}}
	host, _, cards := vaultAndCards(t, agent)
	assembly := container.NewEditorAssembly(container.Config{PluginHost: host}, nil, nil)

	runToEnd(t, assembly.GetAgent(), agentport.Task{Question: "q", Agent: "alpha", Vault: "v-garden"})

	if requests := cards.getRequests(); len(requests) != 1 || requests[0].GetVaultId() != "v-garden" {
		t.Errorf("the plugin was called with %v", requests)
	}
	if request := agent.getRequests()[0]; request.GetVaultId() != "v-garden" {
		t.Errorf("the agent request names the vault %q", request.GetVaultId())
	}
}

// shownVault is a host that exposes one vault and nothing else.
type shownVault struct {
	pluginport.PluginVault
	vault domain.Vault
}

func (s shownVault) GetVault() (domain.Vault, bool) { return s.vault, true }

const mcpPlugin = "md.test.mcp"

// A plugin that serves the tools to programs outside Numen lists the tools of the tools plugins on
// the host, and calls them in the vault the host shows with no context. A tool whose effect it
// holds no permission of is neither listed nor run.
func TestAPluginListsAndCallsTheToolsOfTheOthersThroughTheHost(t *testing.T) {
	agent := &toolAgent{}
	vault := &toolPlugin{tools: []*v1.PluginTool{
		newPluginTool("note_read", v1.ToolEffect_TOOL_EFFECT_READ_ONLY),
		newPluginTool("note_write", v1.ToolEffect_TOOL_EFFECT_READ_WRITE),
	}}
	host := hostOver(t,
		agentAt(agent),
		toolsAt(vaultPlugin, []string{"vault:read", "vault:write"}, vault),
		testPlugin{id: mcpPlugin, permissions: []string{"vault:read"}, mount: func(*http.ServeMux) {}},
	)
	t.Cleanup(func() { _ = host.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond) })
	host.SetPluginVault(shownVault{vault: domain.Vault{ID: "v-garden", Name: "Garden"}})
	served, err := hostendpoint.Open(t.Context(), hostendpoint.Address, host)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = served.Close(context.WithoutCancel(t.Context())) })
	caller, err := host.EnsureActive(t.Context(), mcpPlugin)
	if err != nil {
		t.Fatal(err)
	}
	client := numenv1connect.NewPluginToolServiceClient(
		&http.Client{Transport: bearerTransport{token: caller.Token}}, fmt.Sprintf("http://127.0.0.1:%d", served.GetPort()))

	listed, err := client.ListTools(t.Context(), connect.NewRequest(&v1.ListToolsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, one := range listed.Msg.GetTools() {
		names = append(names, one.GetName())
	}
	if !slices.Equal(names, []string{"note_read"}) {
		t.Errorf("the host lists %v", names)
	}
	res, err := client.CallTool(t.Context(), connect.NewRequest(&v1.CallToolRequest{
		Name: "note_read", Arguments: []byte(`{"path":"Garden.md"}`), Context: map[string]string{cardKey: "c-1"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if text := res.Msg.GetResult().GetText(); text != `note_read {"path":"Garden.md"} ` {
		t.Errorf("the call answered %q", text)
	}
	if requests := vault.getRequests(); len(requests) != 1 || requests[0].GetVaultId() != "v-garden" || len(requests[0].GetContext()) != 0 {
		t.Errorf("the plugin was called with %v", requests)
	}
	denial, err := client.CallTool(t.Context(), connect.NewRequest(&v1.CallToolRequest{Name: "note_write"}))
	if err != nil {
		t.Fatal(err)
	}
	if denial.Msg.GetError() != v1.ToolErrorCode_TOOL_ERROR_CODE_UNKNOWN_TOOL || len(vault.getRequests()) != 1 {
		t.Errorf("a tool the caller may not run answered %v", denial.Msg)
	}
}
