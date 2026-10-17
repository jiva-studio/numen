package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// startAgent serves the agent over HTTP and answers with a client of it.
func startAgent(t *testing.T, agentSvc *agentServer) numenv1connect.AgentServiceClient {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewAgentServiceHandler(agentSvc))
	testServer := httptest.NewServer(mux)
	t.Cleanup(testServer.Close)
	return numenv1connect.NewAgentServiceClient(http.DefaultClient, testServer.URL)
}

// writeScript writes an executable shell script and answers with its path.
func writeScript(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return path
}

func TestLifecycleServer(t *testing.T) {
	shutdownCh := make(chan struct{})
	agentSvc := &agentServer{}
	life := server.NewLifecycleServer(shutdownCh, server.WithLifecycleConfigure(func(_ context.Context, values map[string]string) error {
		agentSvc.applyConfig(values)
		return nil
	}))

	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(life))

	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	client := numenv1connect.NewPluginLifecycleServiceClient(http.DefaultClient, testServer.URL)
	ctx := context.Background()

	pingResp, err := client.Ping(ctx, connect.NewRequest(&numenv1.PingRequest{}))
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
	if pingResp.Msg.GetStatus() != "ok" {
		t.Errorf("unexpected status: %s", pingResp.Msg.GetStatus())
	}

	_, err = client.Configure(ctx, connect.NewRequest(&numenv1.ConfigureRequest{
		Values: map[string]string{"apiKey": "k", "baseUrl": "http://models.test"},
	}))
	if err != nil {
		t.Fatalf("Configure failed: %v", err)
	}
	if agentSvc.catalogue.key != "k" || agentSvc.catalogue.base != "http://models.test" {
		t.Errorf("the models are asked with %q at %q", agentSvc.catalogue.key, agentSvc.catalogue.base)
	}
}

func TestFetchAnthropicModels(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{"id": "claude-3-5-sonnet", "display_name": "Claude 3.5 Sonnet"}
			]
		}`))
	}))
	defer mockServer.Close()

	ctx := context.Background()
	models, err := fetchAnthropicModels(ctx, "test-key", mockServer.URL)
	if err != nil {
		t.Fatalf("fetchAnthropicModels failed: %v", err)
	}
	if len(models) != 1 || models[0].Name != "claude-3-5-sonnet" {
		t.Errorf("unexpected models: %+v", models)
	}
}

// The models are asked for once and kept until the key or the endpoint changes.
func TestTheModelsAreKeptUntilTheKeyChanges(t *testing.T) {
	asked := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		asked++
		_, _ = w.Write([]byte(`{"data": [{"id": "opus", "display_name": "Opus"}]}`))
	}))
	defer mockServer.Close()

	var catalogue modelCatalogue
	catalogue.set("k", mockServer.URL)
	for range 2 {
		if got := catalogue.get(t.Context()); len(got) != 1 || got[0].Name != "opus" {
			t.Fatalf("the models are %+v", got)
		}
	}
	if asked != 1 {
		t.Errorf("the endpoint was asked %d times, want 1", asked)
	}

	catalogue.set("other", mockServer.URL)
	catalogue.get(t.Context())
	if asked != 2 {
		t.Errorf("a new key asked the endpoint %d times in all, want 2", asked)
	}
}

// A request that ends before the endpoint answers keeps nothing, so the next one asks again.
func TestAnEndedRequestKeepsNoModels(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data": [{"id": "opus"}]}`))
	}))
	defer mockServer.Close()

	var catalogue modelCatalogue
	catalogue.set("k", mockServer.URL)
	ended, cancel := context.WithCancel(t.Context())
	cancel()
	if got := catalogue.get(ended); got != nil {
		t.Errorf("an ended request got %+v", got)
	}
	if got := catalogue.get(t.Context()); len(got) != 1 {
		t.Errorf("the next request got %+v", got)
	}
}

func TestAgentServiceRPC(t *testing.T) {
	client := startAgent(t, &agentServer{
		systemPrompt: "Base prompt.",
		tools: []*numenv1.AgentToolDefinition{
			{Name: "search_tool", Description: "Search notes"},
		},
	})
	ctx := context.Background()

	regResp, err := client.RegisterTool(ctx, connect.NewRequest(&numenv1.RegisterToolRequest{
		Tool: &numenv1.AgentToolDefinition{Name: "new_tool", Description: "New tool"},
	}))
	if err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}
	if !regResp.Msg.GetSuccess() {
		t.Errorf("expected success true")
	}

	extResp, err := client.ExtendSystemPrompt(ctx, connect.NewRequest(&numenv1.ExtendSystemPromptRequest{
		Extension: " Extra instructions.",
	}))
	if err != nil {
		t.Fatalf("ExtendSystemPrompt failed: %v", err)
	}
	if extResp.Msg.GetFullPrompt() != "Base prompt. Extra instructions." {
		t.Errorf("unexpected full prompt: %q", extResp.Msg.GetFullPrompt())
	}

	toolsResp, err := client.GetAgentTools(ctx, connect.NewRequest(&numenv1.GetAgentToolsRequest{}))
	if err != nil {
		t.Fatalf("GetAgentTools failed: %v", err)
	}
	if len(toolsResp.Msg.GetTools()) != 2 {
		t.Errorf("expected 2 tools, got %d", len(toolsResp.Msg.GetTools()))
	}

	finResp, err := client.FinishConversation(ctx, connect.NewRequest(&numenv1.FinishConversationRequest{
		Conversation: "conv-1",
	}))
	if err != nil {
		t.Fatalf("FinishConversation failed: %v", err)
	}
	if finResp == nil {
		t.Errorf("expected non-nil response")
	}
}

// The state says the program is out of reach where it is nowhere to be found, and the program list
// says it is not installed.
func TestTheProgramIsUnreachableWhereItIsNotInstalled(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("NUMEN_CLAUDE_COMMAND", "")
	t.Setenv("CLAUDE_COMMAND", "")
	t.Setenv(anthropicKeyVar, "")
	client := startAgent(t, &agentServer{})

	state, err := client.GetAgentState(t.Context(), connect.NewRequest(&numenv1.GetAgentStateRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if state.Msg.GetUnreachable() == "" {
		t.Error("an agent that is not installed was said to be reachable")
	}

	programs, err := client.GetAgentPrograms(t.Context(), connect.NewRequest(&numenv1.GetAgentProgramsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(programs.Msg.GetPrograms()) != 1 || programs.Msg.GetPrograms()[0].GetIsInstalled() {
		t.Errorf("the programs are %v", programs.Msg.GetPrograms())
	}
}

// The program list names Claude Code as the default program, installed where it is found, with the
// models the endpoint lists.
func TestTheProgramListNamesClaudeWithItsModels(t *testing.T) {
	t.Setenv("NUMEN_CLAUDE_COMMAND", writeScript(t, "claude", ""))
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data": [{"id": "opus", "display_name": "Opus"}, {"id": "haiku"}]}`))
	}))
	defer mockServer.Close()

	agentSvc := &agentServer{}
	agentSvc.applyConfig(map[string]string{"apiKey": "k", "baseUrl": mockServer.URL})
	client := startAgent(t, agentSvc)

	got, err := client.GetAgentPrograms(t.Context(), connect.NewRequest(&numenv1.GetAgentProgramsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	programs := got.Msg.GetPrograms()
	if len(programs) != 1 {
		t.Fatalf("the programs are %v", programs)
	}
	program := programs[0]
	if program.GetName() != "claude" || program.GetTitle() != "Claude Code" ||
		!program.GetIsInstalled() || !program.GetIsDefault() {
		t.Errorf("the program is %v", program)
	}
	var names []string
	for _, one := range program.GetModels() {
		names = append(names, one.GetName())
	}
	if !slices.Equal(names, []string{"opus", "haiku"}) {
		t.Errorf("the models are %v", names)
	}
}

// readSteps reads a run to its end and words each step.
func readSteps(stream *connect.ServerStreamForClient[numenv1.AskAgentResponse]) ([]string, error) {
	var steps []string
	for stream.Receive() {
		switch s := stream.Msg().GetStep().(type) {
		case *numenv1.AskAgentResponse_ModelStarted:
			steps = append(steps, "model_started")
		case *numenv1.AskAgentResponse_ToolCall:
			steps = append(steps, "tool_call:"+s.ToolCall.GetTool())
		case *numenv1.AskAgentResponse_ToolFinished:
			steps = append(steps, "tool_finished")
		case *numenv1.AskAgentResponse_Said:
			steps = append(steps, "said:"+s.Said)
		case *numenv1.AskAgentResponse_Stopped:
			steps = append(steps, "stopped:"+s.Stopped)
		}
	}
	return steps, stream.Err()
}

// collectSteps is readSteps, failing the test on an error.
func collectSteps(t *testing.T, stream *connect.ServerStreamForClient[numenv1.AskAgentResponse]) []string {
	t.Helper()
	steps, err := readSteps(stream)
	if err != nil {
		t.Fatalf("stream error: %v", err)
	}
	return steps
}

func TestAskAgentWithMockScript(t *testing.T) {
	script := writeScript(t, "mock-claude.sh", "cat << 'JSONEOF'\n"+
		"{\"type\":\"system\",\"subtype\":\"status\",\"status\":\"requesting\"}\n"+
		"{\"type\":\"stream_event\",\"event\":{\"type\":\"content_block_start\",\"content_block\":{\"type\":\"tool_use\",\"id\":\"call_1\",\"name\":\"mcp__numen__note_search\"}}}\n"+
		"{\"type\":\"stream_event\",\"event\":{\"type\":\"content_block_delta\",\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"{\\\"query\\\":\\\"algebra\\\",\\\"path\\\":\\\"notes/math.md\\\"}\"}}}\n"+
		"{\"type\":\"stream_event\",\"event\":{\"type\":\"content_block_stop\"}}\n"+
		"{\"type\":\"user\",\"message\":{\"content\":[{\"type\":\"tool_result\"}]}}\n"+
		"{\"type\":\"stream_event\",\"event\":{\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"Algebra notes found.\"}}}\n"+
		"{\"type\":\"result\",\"is_error\":false}\n"+
		"JSONEOF\n")
	client := startAgent(t, &agentServer{})

	stream, err := client.AskAgent(t.Context(), connect.NewRequest(&numenv1.AskAgentRequest{
		Prompt:  "Find algebra notes",
		Command: []string{script},
	}))
	if err != nil {
		t.Fatalf("AskAgent failed: %v", err)
	}

	expected := []string{
		"model_started",
		"tool_call:note_search",
		"tool_call:note_search",
		"tool_finished",
		"said:Algebra notes found.",
		"stopped:done",
	}
	if got := collectSteps(t, stream); !slices.Equal(got, expected) {
		t.Errorf("got %v, want %v", got, expected)
	}
}

// The program is started in the folder the task names, with the arguments the request gives it, the
// question on its input, and the tools it is given in a file that is gone when it ends.
func TestAskAgentStartsTheProgramFromTheRequest(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(t.TempDir(), "run")
	script := writeScript(t, "mock-claude.sh",
		"pwd > "+out+".dir\n"+
			"cat > "+out+".stdin\n"+
			"for a in \"$@\"; do printf '%s\\n' \"$a\"; done > "+out+".argv\n"+
			"while [ $# -gt 0 ]; do if [ \"$1\" = --mcp-config ]; then cp \"$2\" "+out+".config; fi; shift; done\n"+
			"echo '{\"type\":\"result\",\"is_error\":false}'\n")
	client := startAgent(t, &agentServer{})

	stream, err := client.AskAgent(t.Context(), connect.NewRequest(&numenv1.AskAgentRequest{
		Prompt:              "the question",
		Command:             []string{script, "--first"},
		Root:                root,
		Model:               "opus",
		MaxSteps:            7,
		Tools:               []string{"note_read"},
		ReadsHooksAndSkills: true,
		ToolsUrl:            "http://127.0.0.1:1/mcp",
		ToolsToken:          "secret",
	}))
	if err != nil {
		t.Fatal(err)
	}
	collectSteps(t, stream)

	read := func(suffix string) string {
		t.Helper()
		raw, err := os.ReadFile(out + suffix)
		if err != nil {
			t.Fatal(err)
		}
		return string(raw)
	}
	if got := strings.TrimSpace(read(".dir")); got != root {
		resolved, _ := filepath.EvalSymlinks(root)
		if got != resolved {
			t.Errorf("started in %q, want %q", got, root)
		}
	}
	if got := read(".stdin"); got != "the question" {
		t.Errorf("the input is %q", got)
	}
	argv := strings.Split(strings.TrimSpace(read(".argv")), "\n")
	if argv[0] != "--first" {
		t.Errorf("the arguments of the command line come first: %v", argv)
	}
	assertPair(t, argv, "--model", "opus")
	assertPair(t, argv, "--max-turns", "7")
	assertPair(t, argv, "--setting-sources", "user")
	assertPair(t, argv, "--allowedTools", "WebSearch,mcp__numen__note_read")
	if config := read(".config"); !strings.Contains(config, "http://127.0.0.1:1/mcp") ||
		!strings.Contains(config, "Bearer secret") {
		t.Errorf("the tools file is %q", config)
	}
}

// A task asked again in the conversation resumes the session the first one reported.
func TestAConversationResumesItsSession(t *testing.T) {
	out := filepath.Join(t.TempDir(), "argv")
	script := writeScript(t, "mock-claude.sh",
		"for a in \"$@\"; do printf '%s\\n' \"$a\"; done > "+out+"\n"+
			"echo '{\"type\":\"system\",\"subtype\":\"init\",\"session_id\":\"s-1\"}'\n")
	client := startAgent(t, &agentServer{})

	ask := func() []string {
		stream, err := client.AskAgent(t.Context(), connect.NewRequest(&numenv1.AskAgentRequest{
			Prompt: "q", Command: []string{script}, Conversation: "c-1",
		}))
		if err != nil {
			t.Fatal(err)
		}
		collectSteps(t, stream)
		raw, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		return strings.Split(strings.TrimSpace(string(raw)), "\n")
	}

	if first := ask(); slices.Contains(first, "--resume") {
		t.Errorf("the first task resumed a session: %v", first)
	}
	assertPair(t, ask(), "--resume", "s-1")

	if _, err := client.FinishConversation(t.Context(), connect.NewRequest(
		&numenv1.FinishConversationRequest{Conversation: "c-1"})); err != nil {
		t.Fatal(err)
	}
	if again := ask(); slices.Contains(again, "--resume") {
		t.Errorf("a finished conversation resumed a session: %v", again)
	}
}

// Finishing a conversation ends the program a task of it is running, and the task ends without a
// reason.
func TestFinishingAConversationStopsItsRun(t *testing.T) {
	script := writeScript(t, "mock-claude.sh", "sleep 60\n")
	client := startAgent(t, &agentServer{})

	stream, err := client.AskAgent(t.Context(), connect.NewRequest(&numenv1.AskAgentRequest{
		Prompt: "q", Command: []string{script}, Conversation: "c-1",
	}))
	if err != nil {
		t.Fatal(err)
	}
	ended := make(chan []string, 1)
	go func() {
		steps, _ := readSteps(stream)
		ended <- steps
	}()

	// The run is registered once the request arrives; finishing until none is left stops it.
	deadline := time.After(10 * time.Second)
	for {
		if _, err := client.FinishConversation(t.Context(), connect.NewRequest(
			&numenv1.FinishConversationRequest{Conversation: "c-1"})); err != nil {
			t.Fatal(err)
		}
		select {
		case steps := <-ended:
			for _, step := range steps {
				if strings.HasPrefix(step, "stopped:") {
					t.Errorf("a stopped run reported %q", step)
				}
			}
			return
		case <-time.After(50 * time.Millisecond):
		case <-deadline:
			t.Fatal("the run was not stopped")
		}
	}
}

// A program nowhere to be found is told to the person, and the run ends.
func TestAskAgentSaysWhenTheProgramIsNotInstalled(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("NUMEN_CLAUDE_COMMAND", "")
	t.Setenv("CLAUDE_COMMAND", "")
	client := startAgent(t, &agentServer{})

	stream, err := client.AskAgent(t.Context(), connect.NewRequest(&numenv1.AskAgentRequest{Prompt: "q"}))
	if err != nil {
		t.Fatal(err)
	}
	steps := collectSteps(t, stream)
	if len(steps) != 2 || !strings.HasPrefix(steps[0], "said:") || steps[1] != "stopped:claude not installed" {
		t.Errorf("the steps are %v", steps)
	}
}

// A closed agent takes no task.
func TestAClosedAgentTakesNoTask(t *testing.T) {
	agentSvc := &agentServer{}
	_ = agentSvc.registry.Close()
	client := startAgent(t, agentSvc)

	stream, err := client.AskAgent(t.Context(), connect.NewRequest(&numenv1.AskAgentRequest{
		Prompt: "q", Command: []string{"true"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	for stream.Receive() {
	}
	if stream.Err() == nil {
		t.Error("a closed agent took a task")
	}
}

// A command line the installation set counts as the program being installed, wherever the program
// stands on this machine.
func TestTheProgramIsInstalledUnderTheCommandLineTheInstallationSet(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("NUMEN_CLAUDE_COMMAND", "")
	t.Setenv("CLAUDE_COMMAND", "")
	t.Setenv(anthropicKeyVar, "")
	client := startAgent(t, &agentServer{})

	got, err := client.GetAgentPrograms(t.Context(), connect.NewRequest(&numenv1.GetAgentProgramsRequest{
		Commands: []*numenv1.AgentProgramCommand{
			{Program: "codex", Command: []string{"/bin/codex"}},
			{Program: "claude", Command: []string{"/opt/claude/bin/claude"}},
		},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if programs := got.Msg.GetPrograms(); len(programs) != 1 || !programs[0].GetIsInstalled() {
		t.Errorf("the programs are %v", programs)
	}
}
