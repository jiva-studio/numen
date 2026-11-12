package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/rpcerror"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/internal/adapter/executable"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/internal/adapter/hostenv"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/model"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/ports"
)

// steppingClock advances by step on every reading.
type steppingClock struct {
	tick time.Time
	step time.Duration
}

func (c *steppingClock) Now() time.Time {
	c.tick = c.tick.Add(c.step)
	return c.tick
}

func newTestAgentWith(t *testing.T, env ports.Environment) *agent {
	t.Helper()
	subject, err := newAgent(parts{environment: env, clock: &steppingClock{tick: time.Unix(0, 0), step: time.Second}, isAsync: false})
	if err != nil {
		t.Fatal(err)
	}
	return subject
}

func newTestAgent(t *testing.T) *agent {
	t.Helper()
	return newTestAgentWith(t, hostenv.Process{})
}

func startAgent(t *testing.T, agentSvc *agent) *agentrun.Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(agentrun.NewHandler(agentSvc, agentSvc))
	testServer := httptest.NewServer(mux)
	t.Cleanup(testServer.Close)
	return agentrun.NewClient(http.DefaultClient, testServer.URL)
}

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
	agentSvc := newTestAgent(t)
	handler, err := server.NewLinkedHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "linked-token"}, func(s *server.Server) error {
		s.SetConfigure(func(_ context.Context, values map[string]string, _ server.Secrets) error {
			agentSvc.settings.Apply(values)
			return nil
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	testServer := httptest.NewServer(handler)
	defer testServer.Close()

	token := hostcall.WithBearer("linked-token")
	client := server.NewLifecycleClient(http.DefaultClient, testServer.URL, token)
	ctx := t.Context()

	ping, err := client.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
	if ping.Status != "ok" {
		t.Errorf("unexpected status: %s", ping.Status)
	}

	if _, err = client.Configure(ctx, map[string]string{"model": "m1"}, nil); err != nil {
		t.Fatalf("Configure failed: %v", err)
	}
	if agentSvc.settings.Get().DefaultModel != "m1" {
		t.Errorf("the model that answers a task naming none is %q", agentSvc.settings.Get().DefaultModel)
	}
}

func TestAgentServiceRPC(t *testing.T) {
	subject := newTestAgent(t)
	subject.toolset.ExtendPrompt("Base prompt.")
	subject.toolset.Register(&agentrun.ToolDefinition{Name: "search_tool", Description: "Search notes"})
	client := startAgent(t, subject)
	ctx := t.Context()

	if err := client.RegisterTool(ctx, &agentrun.ToolDefinition{Name: "new_tool", Description: "New tool"}); err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}

	fullPrompt, err := client.ExtendSystemPrompt(ctx, " Extra instructions.")
	if err != nil {
		t.Fatalf("ExtendSystemPrompt failed: %v", err)
	}
	if fullPrompt != "Base prompt. Extra instructions." {
		t.Errorf("unexpected full prompt: %q", fullPrompt)
	}

	toolSet, err := client.GetTools(ctx)
	if err != nil {
		t.Fatalf("GetTools failed: %v", err)
	}
	if len(toolSet.Tools) != 2 || toolSet.Tools[0].Name != "search_tool" || toolSet.Tools[1].Name != "new_tool" {
		t.Errorf("expected [search_tool, new_tool], got %+v", toolSet.Tools)
	}

	if err := client.FinishConversation(ctx, "conv-1"); err != nil {
		t.Fatalf("FinishConversation failed: %v", err)
	}
}

// The state reports the program as unreachable when it cannot be found, and the program list
// reports it as not installed.
func TestTheProgramIsUnreachableWhereItIsNotInstalled(t *testing.T) {
	withoutClaudeAnywhere(t)
	client := startAgent(t, newTestAgent(t))

	state, err := client.GetState(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if state.Unreachable == "" {
		t.Error("an agent that is not installed was said to be reachable")
	}

	programs, err := client.GetPrograms(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(programs) != 1 || programs[0].IsInstalled || len(programs[0].Models) != len(model.GetFallback()) {
		t.Errorf("the programs are %v", programs)
	}
}

// The program list names Claude Code as the default program, installed where it is found, with the
// models the program accepts, listing with no key and no network.
func TestTheProgramListNamesClaudeWithItsModels(t *testing.T) {
	t.Setenv("NUMEN_CLAUDE_COMMAND", writeScript(t, "claude", ""))
	client := startAgent(t, newTestAgent(t))

	programs, err := client.GetPrograms(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(programs) != 1 {
		t.Fatalf("the programs are %v", programs)
	}
	program := programs[0]
	if program.Name != "claude" || program.Title != "Claude Code" ||
		!program.IsInstalled || !program.IsDefault || !program.LimitsTools {
		t.Errorf("the program is %v", program)
	}
	var names []string
	for _, one := range program.Models {
		names = append(names, one.Name)
	}
	if !slices.Equal(names, []string{"sonnet", "opus", "haiku"}) {
		t.Errorf("the models are %v", names)
	}
}

func wordStep(msg *agentrun.Response) string {
	switch s := msg.GetStep().(type) {
	case *agentrun.ModelStartedStep:
		return "model_started"
	case *agentrun.ToolCallStep:
		return "tool_call:" + s.ToolCall.Tool
	case *agentrun.ToolFinishedStep:
		return "tool_finished"
	case *agentrun.TextStep:
		return "said:" + s.Text
	case *agentrun.StoppedStep:
		return "stopped:" + s.Stopped
	}
	return ""
}

func readSteps(stream *agentrun.Stream) ([]string, error) {
	var steps []string
	for stream.Receive() {
		if word := wordStep(stream.Msg()); word != "" {
			steps = append(steps, word)
		}
	}
	return steps, stream.Err()
}

func drainSteps(t *testing.T, stream *agentrun.Stream) []string {
	t.Helper()
	steps, err := readSteps(stream)
	if err != nil {
		t.Errorf("stream error: %v", err)
	}
	return steps
}

func collectSteps(t *testing.T, stream *agentrun.Stream) []string {
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
		"{\"type\":\"stream_event\",\"event\":{\"type\":\"content_block_delta\",\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\""+
		"{\\\"query\\\":\\\"algebra\\\",\\\"path\\\":\\\"notes/math.md\\\"}\"}}}\n"+
		"{\"type\":\"stream_event\",\"event\":{\"type\":\"content_block_stop\"}}\n"+
		"{\"type\":\"user\",\"message\":{\"content\":[{\"type\":\"tool_result\"}]}}\n"+
		"{\"type\":\"stream_event\",\"event\":{\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"Algebra notes found.\"}}}\n"+
		"{\"type\":\"result\",\"is_error\":false}\n"+
		"JSONEOF\n")
	client := startAgent(t, newTestAgent(t))

	stream, err := client.Ask(t.Context(), &agentrun.Request{
		Prompt:  "Find algebra notes",
		Command: []string{script},
	})
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
			paths = append(paths, call.Path)
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
	subject := newTestAgent(t)
	subject.settings.Apply(map[string]string{"readsHooksAndSkills": "true"})
	client := startAgent(t, subject)

	stream, err := client.Ask(t.Context(), &agentrun.Request{
		Prompt:     "the question",
		Command:    []string{script, "--first"},
		Root:       root,
		Model:      "opus",
		MaxSteps:   7,
		Tools:      []string{"note_read"},
		ToolsURL:   "http://127.0.0.1:1/mcp",
		ToolsToken: "secret",
	})
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
		resolved, err := filepath.EvalSymlinks(root)
		if err != nil {
			t.Fatal(err)
		}
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

// A second task in the same conversation resumes the session reported by the first.
func TestAConversationResumesItsSession(t *testing.T) {
	out := filepath.Join(t.TempDir(), "argv")
	script := writeScript(t, "mock-claude.sh",
		"for a in \"$@\"; do printf '%s\\n' \"$a\"; done > "+out+"\n"+
			"echo '{\"type\":\"system\",\"subtype\":\"init\",\"session_id\":\"s-1\"}'\n")
	client := startAgent(t, newTestAgent(t))

	ask := func() []string {
		stream, err := client.Ask(t.Context(), &agentrun.Request{
			Prompt: "q", Command: []string{script}, Conversation: "c-1",
		})
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

	if err := client.FinishConversation(t.Context(), "c-1"); err != nil {
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
	client := startAgent(t, newTestAgent(t))

	stream, err := client.Ask(t.Context(), &agentrun.Request{
		Prompt: "q", Command: []string{script}, Conversation: "c-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	awaitUp(t, stream)

	if err := client.FinishConversation(t.Context(), "c-1"); err != nil {
		t.Fatal(err)
	}
	endsWithin(t, "the finished conversation", 30*time.Second, func() {
		for _, step := range drainSteps(t, stream) {
			if strings.HasPrefix(step, "stopped:") {
				t.Errorf("a stopped run reported %q", step)
			}
		}
	})
}

// A program that cannot be found is reported to the user, and the run ends.
func TestAskAgent_NotInstalled(t *testing.T) {
	withoutClaudeAnywhere(t)
	client := startAgent(t, newTestAgent(t))

	stream, err := client.Ask(t.Context(), &agentrun.Request{Prompt: "q"})
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
	agentSvc := newTestAgent(t)
	if err := agentSvc.run.Close(); err != nil {
		t.Fatal(err)
	}
	client := startAgent(t, agentSvc)

	stream, err := client.Ask(t.Context(), &agentrun.Request{
		Prompt: "q", Command: []string{"true"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for stream.Receive() {
	}
	if rpcerror.GetCode(stream.Err()) != rpcerror.Unavailable {
		t.Errorf("expected Unavailable error from closed agent, got %v", stream.Err())
	}
}

// A command line the installation set counts as the program being installed, wherever the program
// relies on this machine.
func TestGetAgentPrograms_ExplicitCommandInstallation(t *testing.T) {
	withoutClaudeAnywhere(t)
	t.Setenv("ANTHROPIC_API_KEY", "")
	client := startAgent(t, newTestAgent(t))

	programs, err := client.GetPrograms(t.Context(), []agentrun.ProgramCommand{
		{Program: "codex", Command: []string{"/bin/codex"}},
		{Program: "claude", Command: []string{"/opt/claude/bin/claude"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(programs) != 1 || !programs[0].IsInstalled {
		t.Errorf("the programs are %v", programs)
	}
}

func withoutClaudeAnywhere(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("NUMEN_CLAUDE_COMMAND", "")
	t.Setenv("CLAUDE_COMMAND", "")
	if found := (executable.Finder{}).Find(""); found != "" {
		t.Skipf("Claude Code is installed at %s", found)
	}
}

func TestGetAgentProgramsIgnoresCommandsForOtherPrograms(t *testing.T) {
	withoutClaudeAnywhere(t)
	t.Setenv("ANTHROPIC_API_KEY", "")
	client := startAgent(t, newTestAgent(t))

	programs, err := client.GetPrograms(t.Context(), []agentrun.ProgramCommand{
		{Program: "other", Command: []string{"/bin/other"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(programs) != 1 || programs[0].IsInstalled {
		t.Errorf("expected claude to not be installed, got programs: %v", programs)
	}
}

// A task that names no tool is one answer: the program is given no tool, whatever tools the server
// holds, and no file naming the vault's tools.
func TestAskAgent_NoToolsWhenRequestToolsEmpty(t *testing.T) {
	out := filepath.Join(t.TempDir(), "run")
	script := writeScript(t, "mock-claude.sh",
		"for a in \"$@\"; do printf '%s\\n' \"$a\"; done > "+out+".argv\n"+
			"echo '{\"type\":\"result\",\"is_error\":false}'\n")
	agentSvc := newTestAgent(t)
	agentSvc.toolset.Register(&agentrun.ToolDefinition{Name: "note_read"})
	agentSvc.toolset.Register(&agentrun.ToolDefinition{Name: "note_write"})
	client := startAgent(t, agentSvc)

	stream, err := client.Ask(t.Context(), &agentrun.Request{
		Prompt:     "q",
		Command:    []string{script},
		ToolsURL:   "http://127.0.0.1:7717/mcp",
		ToolsToken: "let-me-in",
	})
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
	agentSvc := newTestAgent(t)
	agentSvc.settings.Apply(map[string]string{"apiKey": "host-key", "model": "m1"})
	client := startAgent(t, agentSvc)

	stream, err := client.Ask(t.Context(), &agentrun.Request{
		Prompt: "q", Command: []string{script},
	})
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
	agentSvc := newTestAgent(t)
	agentSvc.toolset.ExtendPrompt("vault prompt")
	client := startAgent(t, agentSvc)

	stream, err := client.Ask(t.Context(), &agentrun.Request{
		Prompt:       "q",
		Command:      []string{script},
		Instructions: "task instructions",
	})
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
	client := startAgent(t, newTestAgent(t))

	stream, err := client.Ask(t.Context(), &agentrun.Request{
		Prompt:     "q",
		Command:    []string{script},
		Tools:      []string{"note_read"},
		ToolsURL:   "http://127.0.0.1:1/mcp",
		ToolsToken: "tok",
	})
	if err != nil {
		t.Fatal(err)
	}
	for stream.Receive() {
	}
	if rpcerror.GetCode(stream.Err()) != rpcerror.Internal {
		t.Fatalf("expected CodeInternal when mcp config cannot be written, got: %v", stream.Err())
	}
}

func TestAskAgentDropsInternalClaudeEnvironmentVariables(t *testing.T) {
	t.Setenv("CLAUDECODE", "internal-secret")
	out := filepath.Join(t.TempDir(), "run.env")
	script := writeScript(t, "mock-claude.sh",
		"env > "+out+"\n"+
			"echo '{\"type\":\"result\",\"is_error\":false}'\n")
	client := startAgent(t, newTestAgent(t))

	stream, err := client.Ask(t.Context(), &agentrun.Request{
		Prompt:  "q",
		Command: []string{script},
	})
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

// A Claude Code installation needs no key of the plugin: the models are listing with none, and a key
// handed to the plugin is not the program's key.
func TestTheProgramListNeedsNoKey(t *testing.T) {
	t.Setenv("NUMEN_CLAUDE_COMMAND", writeScript(t, "claude", ""))
	t.Setenv("ANTHROPIC_API_KEY", "")
	agentSvc := newTestAgent(t)
	agentSvc.settings.Apply(map[string]string{})
	client := startAgent(t, agentSvc)

	got, err := client.GetPrograms(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if listing := got[0].Models; len(listing) != len(model.GetFallback()) {
		t.Errorf("the models are %v", listing)
	}
}

func TestServerEdgeCasesAndDirectCalls(t *testing.T) {
	s := newTestAgent(t)

	// Ask when claude is not installed and send fails on the first step returns that error.
	withoutClaudeAnywhere(t)
	sendErr := rpcerror.New(rpcerror.Canceled, os.ErrClosed)
	if err := s.Ask(t.Context(), &agentrun.Request{Prompt: "q"}, func(*agentrun.Response) error {
		return sendErr
	}); err == nil {
		t.Error("expected error when send fails on not-installed message")
	}

	// Ask when send fails on ModelStartedStep returns that error.
	if err := s.Ask(t.Context(), &agentrun.Request{Prompt: "q", Command: []string{"true"}}, func(*agentrun.Response) error {
		return sendErr
	}); err == nil {
		t.Error("expected error when send fails on ModelStartedStep")
	}

	// Ask when command fails to start returns Internal error.
	if err := s.Ask(t.Context(), &agentrun.Request{Prompt: "q", Command: []string{"/nonexistent-claude-bin"}}, func(*agentrun.Response) error {
		return nil
	}); rpcerror.GetCode(err) != rpcerror.Internal {
		t.Errorf("expected Internal error on nonexistent binary, got %v", err)
	}

	// Ask when script writes to stderr and exits non-zero reports stderr failure message.
	failCmd := []string{"/bin/sh", "-c", "echo 'fatal stderr crash' >&2\nexit 1\n", "sh"}
	client := startAgent(t, newTestAgent(t))
	stream, err := client.Ask(t.Context(), &agentrun.Request{Prompt: "q", Command: failCmd})
	if err != nil {
		t.Fatal(err)
	}
	steps := collectSteps(t, stream)
	if !slices.Contains(steps, "stopped:fatal stderr crash") {
		t.Errorf("expected stopped:fatal stderr crash, got %v", steps)
	}

	// Ask with tools (expectsVault=true) vs without tools (expectsVault=false) on empty mcp_servers init.
	initNoVaultCmd := []string{"/bin/sh", "-c", "echo '{\"type\":\"system\",\"subtype\":\"init\",\"mcp_servers\":[]}'\n", "sh"}
	streamNoTools, err := client.Ask(t.Context(), &agentrun.Request{Prompt: "q", Command: initNoVaultCmd})
	if err != nil {
		t.Fatal(err)
	}
	stepsNoTools := collectSteps(t, streamNoTools)
	if !slices.Contains(stepsNoTools, "stopped:done") {
		t.Errorf("expected stopped:done when no tools requested, got %v", stepsNoTools)
	}

	streamWithTools, err := client.Ask(t.Context(), &agentrun.Request{
		Prompt:     "q",
		Command:    initNoVaultCmd,
		Tools:      []string{"note_read"},
		ToolsURL:   "http://127.0.0.1:1/mcp",
		ToolsToken: "tok",
	})
	if err != nil {
		t.Fatal(err)
	}
	stepsWithTools := collectSteps(t, streamWithTools)
	if !slices.Contains(stepsWithTools, "stopped:the agent was started without this vault") {
		t.Errorf("expected vault missing reason when tools requested, got %v", stepsWithTools)
	}
}

// The person's own hooks and skills are read only where the plugin's setting `readsHooksAndSkills`
// is on and the task names a tool; a task carries no such option.
func TestAskReadsHooksAndSkillsOnlyWhereTheSettingIsOn(t *testing.T) {
	for _, c := range []struct {
		name   string
		values map[string]string
		want   string
	}{
		{name: "the setting is on", values: map[string]string{"readsHooksAndSkills": "true"}, want: "user"},
		{name: "the setting is off", values: map[string]string{"readsHooksAndSkills": "false"}, want: ""},
		{name: "the setting is never configured", values: map[string]string{}, want: ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "argv")
			script := writeScript(t, "mock-claude.sh",
				"for a in \"$@\"; do printf '%s\\n' \"$a\"; done > "+out+"\n"+
					"echo '{\"type\":\"result\",\"is_error\":false}'\n")
			subject := newTestAgent(t)
			subject.settings.Apply(c.values)
			client := startAgent(t, subject)

			stream, err := client.Ask(t.Context(), &agentrun.Request{
				Prompt: "q", Command: []string{script}, Tools: []string{"note_read"},
			})
			if err != nil {
				t.Fatal(err)
			}
			collectSteps(t, stream)

			raw, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}
			assertPair(t, strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n"), "--setting-sources", c.want)
		})
	}
}

func assertPair(t *testing.T, args []string, flag, want string) {
	t.Helper()
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			if args[i+1] != want {
				t.Errorf("%s = %q, want %q", flag, args[i+1], want)
			}
			return
		}
	}
	t.Errorf("%s absent from %v", flag, args)
}
