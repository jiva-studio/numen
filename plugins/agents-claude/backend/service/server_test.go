package service

import (
	"context"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
)

// startAgent serves the agent over HTTP and answers with a client of it.
func startAgent(t *testing.T, agentSvc *agentServer) agentrun.AgentServiceClient {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(agentrun.NewAgentServiceHandler(agentSvc))
	testServer := httptest.NewServer(mux)
	t.Cleanup(testServer.Close)
	return agentrun.NewAgentServiceClient(http.DefaultClient, testServer.URL)
}

// writeScript writes an executable shell script and answers with its path.
func writeScript(t *testing.T, name, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fixtures are shell scripts")
	}
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
	mux.Handle(server.NewPluginLifecycleServiceHandler(life))

	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	client := server.NewPluginLifecycleServiceClient(http.DefaultClient, testServer.URL)
	ctx := context.Background()

	pingResp, err := client.Ping(ctx, connect.NewRequest(&server.PingRequest{}))
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
	if pingResp.Msg.GetStatus() != "ok" {
		t.Errorf("unexpected status: %s", pingResp.Msg.GetStatus())
	}

	_, err = client.Configure(ctx, connect.NewRequest(&server.ConfigureRequest{
		Values: map[string]string{"model": "m1"},
	}))
	if err != nil {
		t.Fatalf("Configure failed: %v", err)
	}
	if agentSvc.defaultModel != "m1" {
		t.Errorf("the model that answers a task naming none is %q", agentSvc.defaultModel)
	}
}

func TestAgentServiceRPC(t *testing.T) {
	client := startAgent(t, &agentServer{
		systemPrompt: "Base prompt.",
		tools: []*agentrun.AgentToolDefinition{
			{Name: "search_tool", Description: "Search notes"},
		},
	})
	ctx := context.Background()

	regResp, err := client.RegisterTool(ctx, connect.NewRequest(&agentrun.RegisterToolRequest{
		Tool: &agentrun.AgentToolDefinition{Name: "new_tool", Description: "New tool"},
	}))
	if err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}
	if !regResp.Msg.GetSuccess() {
		t.Errorf("expected success true")
	}

	extResp, err := client.ExtendSystemPrompt(ctx, connect.NewRequest(&agentrun.ExtendSystemPromptRequest{
		Extension: " Extra instructions.",
	}))
	if err != nil {
		t.Fatalf("ExtendSystemPrompt failed: %v", err)
	}
	if extResp.Msg.GetFullPrompt() != "Base prompt. Extra instructions." {
		t.Errorf("unexpected full prompt: %q", extResp.Msg.GetFullPrompt())
	}

	toolsResp, err := client.GetAgentTools(ctx, connect.NewRequest(&agentrun.GetAgentToolsRequest{}))
	if err != nil {
		t.Fatalf("GetAgentTools failed: %v", err)
	}
	if len(toolsResp.Msg.GetTools()) != 2 {
		t.Errorf("expected 2 tools, got %d", len(toolsResp.Msg.GetTools()))
	}

	finResp, err := client.FinishConversation(ctx, connect.NewRequest(&agentrun.FinishConversationRequest{
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
	withoutClaudeAnywhere(t)
	client := startAgent(t, &agentServer{})

	state, err := client.GetAgentState(t.Context(), connect.NewRequest(&agentrun.GetAgentStateRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if state.Msg.GetUnreachable() == "" {
		t.Error("an agent that is not installed was said to be reachable")
	}

	programs, err := client.GetAgentPrograms(t.Context(), connect.NewRequest(&agentrun.GetAgentProgramsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(programs.Msg.GetPrograms()) != 1 || programs.Msg.GetPrograms()[0].GetIsInstalled() {
		t.Errorf("the programs are %v", programs.Msg.GetPrograms())
	}
}

// The program list names Claude Code as the default program, installed where it is found, with the
// models the program accepts, listed with no key and no network.
func TestTheProgramListNamesClaudeWithItsModels(t *testing.T) {
	t.Setenv("NUMEN_CLAUDE_COMMAND", writeScript(t, "claude", ""))
	client := startAgent(t, &agentServer{})

	got, err := client.GetAgentPrograms(t.Context(), connect.NewRequest(&agentrun.GetAgentProgramsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	programs := got.Msg.GetPrograms()
	if len(programs) != 1 {
		t.Fatalf("the programs are %v", programs)
	}
	program := programs[0]
	if program.GetName() != "claude" || program.GetTitle() != "Claude Code" ||
		!program.GetIsInstalled() || !program.GetIsDefault() || !program.GetLimitsTools() {
		t.Errorf("the program is %v", program)
	}
	var names []string
	for _, one := range program.GetModels() {
		names = append(names, one.GetName())
	}
	if !slices.Equal(names, []string{"sonnet", "opus", "haiku"}) {
		t.Errorf("the models are %v", names)
	}
}

// wordStep words one step of a run.
func wordStep(msg *agentrun.AskAgentResponse) string {
	switch s := msg.GetStep().(type) {
	case *agentrun.AskAgentResponse_ModelStarted:
		return "model_started"
	case *agentrun.AskAgentResponse_ToolCall:
		return "tool_call:" + s.ToolCall.GetTool()
	case *agentrun.AskAgentResponse_ToolFinished:
		return "tool_finished"
	case *agentrun.AskAgentResponse_Text:
		return "said:" + s.Text
	case *agentrun.AskAgentResponse_Stopped:
		return "stopped:" + s.Stopped
	}
	return ""
}

// readSteps reads a run to its end and words each step.
func readSteps(stream *connect.ServerStreamForClient[agentrun.AskAgentResponse]) ([]string, error) {
	var steps []string
	for stream.Receive() {
		if word := wordStep(stream.Msg()); word != "" {
			steps = append(steps, word)
		}
	}
	return steps, stream.Err()
}

// collectSteps is readSteps, failing the test on an error.
func collectSteps(t *testing.T, stream *connect.ServerStreamForClient[agentrun.AskAgentResponse]) []string {
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

	stream, err := client.AskAgent(t.Context(), connect.NewRequest(&agentrun.AskAgentRequest{
		Prompt:  "Find algebra notes",
		Command: []string{script},
	}))
	if err != nil {
		t.Fatalf("AskAgent failed: %v", err)
	}

	expected := []string{
		"model_started",
		"model_started",
		"tool_call:note_search",
		"tool_call:note_search",
		"tool_finished",
		"said:Algebra notes found.",
		"stopped:done",
	}
	var got, paths []string
	for stream.Receive() {
		if call := stream.Msg().GetToolCall(); call != nil {
			paths = append(paths, call.GetPath())
		}
		if word := wordStep(stream.Msg()); word != "" {
			got = append(got, word)
		}
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, expected) {
		t.Errorf("got %v, want %v", got, expected)
	}
	if len(paths) != 2 || paths[1] != "notes/math.md" {
		t.Errorf("expected final tool path 'notes/math.md', got %v", paths)
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

	stream, err := client.AskAgent(t.Context(), connect.NewRequest(&agentrun.AskAgentRequest{
		Prompt:     "the question",
		Command:    []string{script, "--first"},
		Root:       root,
		Model:      "opus",
		MaxSteps:   7,
		Tools:      []string{"note_read"},
		Options:    map[string]string{"reads_hooks_and_skills": "true"},
		ToolsUrl:   "http://127.0.0.1:1/mcp",
		ToolsToken: "secret",
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
		stream, err := client.AskAgent(t.Context(), connect.NewRequest(&agentrun.AskAgentRequest{
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
		&agentrun.FinishConversationRequest{Conversation: "c-1"})); err != nil {
		t.Fatal(err)
	}
	if again := ask(); slices.Contains(again, "--resume") {
		t.Errorf("a finished conversation resumed a session: %v", again)
	}
}

// Finishing a conversation ends the program a task of it is running, and the task ends without a
// reason.
func TestFinishingAConversationStopsItsRun(t *testing.T) {
	script := writeScript(t, "mock-claude.sh", announce+"sleep 120\n")
	client := startAgent(t, &agentServer{})

	stream, err := client.AskAgent(t.Context(), connect.NewRequest(&agentrun.AskAgentRequest{
		Prompt: "q", Command: []string{script}, Conversation: "c-1",
	}))
	if err != nil {
		t.Fatal(err)
	}
	awaitUp(t, stream)

	if _, err := client.FinishConversation(t.Context(), connect.NewRequest(
		&agentrun.FinishConversationRequest{Conversation: "c-1"})); err != nil {
		t.Fatal(err)
	}
	endsWithin(t, "the finished conversation", 30*time.Second, func() {
		steps, _ := readSteps(stream)
		for _, step := range steps {
			if strings.HasPrefix(step, "stopped:") {
				t.Errorf("a stopped run reported %q", step)
			}
		}
	})
}

// A program nowhere to be found is told to the person, and the run ends.
func TestAskAgent_NotInstalled(t *testing.T) {
	withoutClaudeAnywhere(t)
	client := startAgent(t, &agentServer{})

	stream, err := client.AskAgent(t.Context(), connect.NewRequest(&agentrun.AskAgentRequest{Prompt: "q"}))
	if err != nil {
		t.Fatal(err)
	}
	steps := collectSteps(t, stream)
	if len(steps) != 2 || !strings.HasPrefix(steps[0], "said:") || steps[1] != "stopped:claude not installed" {
		t.Errorf("the steps are %v", steps)
	}
}

// A closed agent takes no task.
func TestAskAgent_ClosedAgent(t *testing.T) {
	agentSvc := &agentServer{}
	_ = agentSvc.registry.Close()
	client := startAgent(t, agentSvc)

	stream, err := client.AskAgent(t.Context(), connect.NewRequest(&agentrun.AskAgentRequest{
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
// relies on this machine.
func TestGetAgentPrograms_ExplicitCommandInstallation(t *testing.T) {
	withoutClaudeAnywhere(t)
	t.Setenv("ANTHROPIC_API_KEY", "")
	client := startAgent(t, &agentServer{})

	got, err := client.GetAgentPrograms(t.Context(), connect.NewRequest(&agentrun.GetAgentProgramsRequest{
		Commands: []*agentrun.AgentProgramCommand{
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

// withoutClaudeAnywhere makes this machine hold no Claude Code, wherever the program is searched
// for.
func withoutClaudeAnywhere(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("NUMEN_CLAUDE_COMMAND", "")
	t.Setenv("CLAUDE_COMMAND", "")
	was := claudeSearchPlaces
	claudeSearchPlaces = nil
	t.Cleanup(func() { claudeSearchPlaces = was })
}

func TestGetAgentProgramsIgnoresCommandsForOtherPrograms(t *testing.T) {
	withoutClaudeAnywhere(t)
	t.Setenv("ANTHROPIC_API_KEY", "")
	client := startAgent(t, &agentServer{})

	got, err := client.GetAgentPrograms(t.Context(), connect.NewRequest(&agentrun.GetAgentProgramsRequest{
		Commands: []*agentrun.AgentProgramCommand{
			{Program: "other", Command: []string{"/bin/other"}},
		},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if programs := got.Msg.GetPrograms(); len(programs) != 1 || programs[0].GetIsInstalled() {
		t.Errorf("expected claude to not be installed, got programs: %v", programs)
	}
}

// A task that names no tool is one answer: the program is given no tool, whatever tools the server
// holds, and no file naming the vault's tools.
func TestAskAgentOffersNoToolWhenRequestToolsEmpty(t *testing.T) {
	out := filepath.Join(t.TempDir(), "run")
	script := writeScript(t, "mock-claude.sh",
		"for a in \"$@\"; do printf '%s\\n' \"$a\"; done > "+out+".argv\n"+
			"echo '{\"type\":\"result\",\"is_error\":false}'\n")
	agentSvc := &agentServer{
		tools: []*agentrun.AgentToolDefinition{
			{Name: "note_read"},
			{Name: "note_write"},
		},
	}
	client := startAgent(t, agentSvc)

	stream, err := client.AskAgent(t.Context(), connect.NewRequest(&agentrun.AskAgentRequest{
		Prompt:     "q",
		Command:    []string{script},
		ToolsUrl:   "http://127.0.0.1:7717/mcp",
		ToolsToken: "let-me-in",
	}))
	if err != nil {
		t.Fatal(err)
	}
	collectSteps(t, stream)

	raw, err := os.ReadFile(out + ".argv")
	if err != nil {
		t.Fatal(err)
	}
	argv := strings.Split(strings.TrimSpace(string(raw)), "\n")
	assertPair(t, argv, "--tools", "")
	assertPair(t, argv, "--max-turns", "1")
	for _, flag := range []string{"--allowedTools", "--mcp-config"} {
		if slices.Contains(argv, flag) {
			t.Errorf("%s is among %v", flag, argv)
		}
	}
	for _, arg := range argv {
		if strings.Contains(arg, "note_") || strings.Contains(arg, "let-me-in") || strings.Contains(arg, "WebSearch") {
			t.Errorf("the program was given %q", arg)
		}
	}
}

// The program runs with the environment of the plugin currently, and the model the host
// configured answers a task that names none.
func TestAskAgentRunsWithTheEnvironmentAndTheConfiguredModel(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "env-key")
	out := filepath.Join(t.TempDir(), "run")
	script := writeScript(t, "mock-claude.sh",
		"echo \"key=$ANTHROPIC_API_KEY\" > "+out+"\n"+
			"for a in \"$@\"; do printf '%s\\n' \"$a\"; done >> "+out+"\n"+
			"echo '{\"type\":\"result\",\"is_error\":false}'\n")
	agentSvc := &agentServer{}
	agentSvc.applyConfig(map[string]string{"apiKey": "host-key", "model": "m1"})
	client := startAgent(t, agentSvc)

	stream, err := client.AskAgent(t.Context(), connect.NewRequest(&agentrun.AskAgentRequest{
		Prompt: "q", Command: []string{script},
	}))
	if err != nil {
		t.Fatal(err)
	}
	collectSteps(t, stream)

	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	argv := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if argv[0] != "key=env-key" {
		t.Errorf("the program saw %q", argv[0])
	}
	assertPair(t, argv, "--model", "m1")
}

func TestAskAgentAppendsSystemPrompt(t *testing.T) {
	out := filepath.Join(t.TempDir(), "run")
	script := writeScript(t, "mock-claude.sh",
		"for a in \"$@\"; do printf '%s\\0' \"$a\"; done > "+out+".argv\n"+
			"echo '{\"type\":\"result\",\"is_error\":false}'\n")
	agentSvc := &agentServer{systemPrompt: "vault prompt"}
	client := startAgent(t, agentSvc)

	stream, err := client.AskAgent(t.Context(), connect.NewRequest(&agentrun.AskAgentRequest{
		Prompt:       "q",
		Command:      []string{script},
		Instructions: "task instructions",
	}))
	if err != nil {
		t.Fatal(err)
	}
	collectSteps(t, stream)

	raw, err := os.ReadFile(out + ".argv")
	if err != nil {
		t.Fatal(err)
	}
	argv := strings.Split(string(raw), "\x00")
	assertPair(t, argv, "--append-system-prompt", "task instructions\nvault prompt")
}

func TestAskAgent_McpConfigWriteFailure(t *testing.T) {
	script := writeScript(t, "mock-claude.sh", "echo '{\"type\":\"result\",\"is_error\":false}'\n")
	t.Setenv("TMPDIR", "/dev/null/nowhere")
	client := startAgent(t, &agentServer{})

	stream, err := client.AskAgent(t.Context(), connect.NewRequest(&agentrun.AskAgentRequest{
		Prompt:     "q",
		Command:    []string{script},
		Tools:      []string{"note_read"},
		ToolsUrl:   "http://127.0.0.1:1/mcp",
		ToolsToken: "tok",
	}))
	if err != nil {
		t.Fatal(err)
	}
	for stream.Receive() {
	}
	if connect.CodeOf(stream.Err()) != connect.CodeInternal {
		t.Fatalf("expected CodeInternal when mcp config cannot be written, got: %v", stream.Err())
	}
}

func TestAskAgentDropsInternalClaudeEnvironmentVariables(t *testing.T) {
	t.Setenv("CLAUDECODE", "internal-secret")
	out := filepath.Join(t.TempDir(), "run.env")
	script := writeScript(t, "mock-claude.sh",
		"env > "+out+"\n"+
			"echo '{\"type\":\"result\",\"is_error\":false}'\n")
	client := startAgent(t, &agentServer{})

	stream, err := client.AskAgent(t.Context(), connect.NewRequest(&agentrun.AskAgentRequest{
		Prompt:  "q",
		Command: []string{script},
	}))
	if err != nil {
		t.Fatal(err)
	}
	collectSteps(t, stream)

	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "CLAUDECODE") {
		t.Fatalf("environment contains CLAUDECODE:\n%s", raw)
	}
}

// A Claude Code installation needs no key of the plugin: the models are listed with none, and a key
// handed to the plugin is not the program's key.
func TestTheProgramListNeedsNoKey(t *testing.T) {
	t.Setenv("NUMEN_CLAUDE_COMMAND", writeScript(t, "claude", ""))
	t.Setenv("ANTHROPIC_API_KEY", "")
	agentSvc := &agentServer{}
	agentSvc.applyConfig(map[string]string{})
	client := startAgent(t, agentSvc)

	got, err := client.GetAgentPrograms(t.Context(), connect.NewRequest(&agentrun.GetAgentProgramsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if models := got.Msg.GetPrograms()[0].GetModels(); len(models) != len(fallbackModels) {
		t.Errorf("the models are %v", models)
	}
}
