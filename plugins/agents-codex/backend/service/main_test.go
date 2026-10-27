package service

import (
	"context"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun/agenttest"

	"connectrpc.com/connect"
)

// The program list names Codex, installed where it is found, with the models it lists under the command
// line the installation set for it.
func TestTheProgramListNamesCodexWithTheModelsItLists(t *testing.T) {
	agenttest.NeedsShell(t)
	t.Setenv("HOME", t.TempDir())
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "codex"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	appServer := agenttest.WriteScript(t, "codex",
		`echo '{"id":1,"result":{}}'`+"\n"+
			`echo '{"id":2,"result":{"data":[{"model":"gpt-x","displayName":"GPT X","isDefault":true},{"model":"hid","hidden":true}]}}'`+"\n"+
			"while read -r _; do :; done\n")
	client := agenttest.Serve(t, &agentServer{})

	got, err := client.GetAgentPrograms(t.Context(), connect.NewRequest(&agentrun.GetAgentProgramsRequest{
		Commands: []*agentrun.AgentProgramCommand{{Program: "codex", Command: []string{appServer}}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	programs := got.Msg.GetPrograms()
	if len(programs) != 1 {
		t.Fatalf("the programs are %v", programs)
	}
	codex := programs[0]
	if codex.GetName() != "codex" || codex.GetTitle() != "OpenAI Codex" || !codex.GetIsInstalled() || codex.GetIsDefault() {
		t.Errorf("the programs are %v", programs)
	}
	if models := codex.GetModels(); len(models) != 1 || models[0].GetName() != "gpt-x" ||
		models[0].GetTitle() != "GPT X" || !models[0].GetIsDefault() {
		t.Errorf("Codex lists %v", codex.GetModels())
	}
}

// A program that is not installed is listed so, and Codex is not started for its models.
func TestAProgramThatIsNotInstalledListsNoModels(t *testing.T) {
	withoutAgentsAnywhere(t)
	client := agenttest.Serve(t, &agentServer{})

	got, err := client.GetAgentPrograms(t.Context(), connect.NewRequest(&agentrun.GetAgentProgramsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range got.Msg.GetPrograms() {
		if one.GetIsInstalled() || len(one.GetModels()) != 0 {
			t.Errorf("%v", one)
		}
	}
}

// What Codex listed is cached for as long as the command line stays the same.
func TestCodexModelCache_Retention(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	counter := filepath.Join(t.TempDir(), "starts")
	command := agenttest.WriteScript(t, "codex",
		"echo x >> "+counter+"\n"+
			`echo '{"id":1,"result":{}}'`+"\n"+
			`echo '{"id":2,"result":{"data":[{"model":"gpt-x"}]}}'`+"\n"+
			"while read -r _; do :; done\n")
	var cache codexModelCache

	for range 2 {
		if got := cache.get(t.Context(), []string{command}); len(got) != 1 || got[0].Name != "gpt-x" {
			t.Fatalf("Codex lists %+v", got)
		}
	}
	raw, err := os.ReadFile(counter)
	if err != nil {
		t.Fatal(err)
	}
	if starts := strings.Count(string(raw), "x"); starts != 1 {
		t.Errorf("Codex was started %d times, want 1", starts)
	}
}

// A request that ended before Codex answered keeps nothing.
func TestAnEndedRequestKeepsNoCodexModels(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	command := agenttest.WriteScript(t, "codex",
		`echo '{"id":1,"result":{}}'`+"\n"+
			`echo '{"id":2,"result":{"data":[{"model":"gpt-x"}]}}'`+"\n"+
			"while read -r _; do :; done\n")
	var cache codexModelCache

	ended, cancel := context.WithCancel(t.Context())
	cancel()
	if got := cache.get(ended, []string{command}); got != nil {
		t.Errorf("an ended request got %+v", got)
	}
	if got := cache.get(t.Context(), []string{command}); len(got) != 1 {
		t.Errorf("the next request got %+v", got)
	}
}

// A command line the installation set counts as the program being installed, and its models are
// asked of that command.
func TestGetAgentPrograms_ExplicitCommandInstallation(t *testing.T) {
	withoutAgentsAnywhere(t)
	appServer := agenttest.WriteScript(t, "codex",
		`echo '{"id":1,"result":{}}'`+"\n"+
			`echo '{"id":2,"result":{"data":[{"model":"gpt-x"}]}}'`+"\n"+
			"while read -r _; do :; done\n")
	client := agenttest.Serve(t, &agentServer{})

	got, err := client.GetAgentPrograms(t.Context(), connect.NewRequest(&agentrun.GetAgentProgramsRequest{
		Commands: []*agentrun.AgentProgramCommand{{Program: "codex", Command: []string{appServer}}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range got.Msg.GetPrograms() {
		if !one.GetIsInstalled() || len(one.GetModels()) != 1 {
			t.Errorf("codex is %v", one)
		}
	}
}

// A listing that failed is left alone for a while and then asked for again, and what Codex lists then is
// cached.
func TestAFailedCodexListingIsAskedForAgain(t *testing.T) {
	agenttest.NeedsShell(t)
	t.Setenv("HOME", t.TempDir())
	failing := agenttest.WriteScript(t, "codex", "exit 1\n")
	now := time.Unix(1000, 0)
	cache := codexModelCache{now: func() time.Time { return now }}

	if got := cache.get(t.Context(), []string{failing}); got != nil {
		t.Fatalf("a failed listing gave %+v", got)
	}
	// The same command line, working now.
	if err := os.WriteFile(failing, []byte("#!/bin/sh\n"+
		`echo '{"id":1,"result":{}}'`+"\n"+
		`echo '{"id":2,"result":{"data":[{"model":"gpt-x"}]}}'`+"\n"+
		"while read -r _; do :; done\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := cache.get(t.Context(), []string{failing}); got != nil {
		t.Fatalf("a command line that just failed was asked again: %+v", got)
	}
	now = now.Add(codexFailureBackoff)
	if got := cache.get(t.Context(), []string{failing}); len(got) != 1 || got[0].Name != "gpt-x" {
		t.Errorf("the next listing gave %+v", got)
	}
}

// withoutAgentsAnywhere makes this machine hold none of the programs, wherever they are searched for.
func withoutAgentsAnywhere(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	was := codexPlaces
	codexPlaces = nil
	t.Cleanup(func() { codexPlaces = was })
}

// The run is started from the request: the command line, the folder, the model and the tools it is
// given.
func TestAskAgentStartsTheProgramFromTheRequest(t *testing.T) {
	root := t.TempDir()
	script, read := agenttest.RecordingProgram(t, "codex")

	agenttest.Ask(t, agenttest.Serve(t, &agentServer{}), &agentrun.AskAgentRequest{
		Prompt:     "the question",
		Command:    []string{script, "--first"},
		Root:       root,
		Model:      "gpt-x",
		Tools:      []string{"note_read"},
		ToolsUrl:   "http://127.0.0.1:1/mcp",
		ToolsToken: "secret",
	})

	dir, lines := read()
	if lines[0] != "--first" || lines[1] != "exec" || lines[len(lines)-1] != "the question" {
		t.Errorf("the arguments are %v", lines)
	}
	for _, want := range []string{"--model", "gpt-x", `mcp_servers.numen.url="http://127.0.0.1:1/mcp"`} {
		if !slices.Contains(lines, want) {
			t.Errorf("%q absent from %v", want, lines)
		}
	}
	if !agenttest.IsDir(dir, root) {
		t.Errorf("started in %q, want %q", dir, root)
	}
}

// A program that fails is reported to the stream as the reason the run stopped.
func TestAskAgentReportsAFailedRun(t *testing.T) {
	script := agenttest.WriteScript(t, "codex", "echo 'not signed in' >&2\nexit 2\n")
	stream, err := agenttest.Serve(t, &agentServer{}).AskAgent(t.Context(), connect.NewRequest(&agentrun.AskAgentRequest{
		Command: []string{script},
	}))
	if err != nil {
		t.Fatal(err)
	}
	var stopReason string
	for stream.Receive() {
		if reason := stream.Msg().GetStopped(); reason != "" {
			stopReason = reason
		}
	}
	if stopReason != "not signed in" {
		t.Errorf("the run stopped with %q", stopReason)
	}
}

// A request naming no tool is one answer: the program is given no MCP server, no sandbox bypass,
// none of its own tools that can be turned off, an empty folder, and the instructions before the
// prompt.
func TestAskAgentOffersNoToolWhenRequestToolsEmpty(t *testing.T) {
	root := t.TempDir()
	script, read := agenttest.RecordingProgram(t, "codex")

	agenttest.Ask(t, agenttest.Serve(t, &agentServer{}), &agentrun.AskAgentRequest{
		Prompt:       "the question",
		Instructions: "be brief",
		Command:      []string{script},
		Root:         root,
		Model:        "gpt-x",
		ToolsUrl:     "http://127.0.0.1:1/mcp",
		ToolsToken:   "secret",
	})

	dir, lines := read()
	joined := strings.Join(lines, "\n")
	if lines[0] != "exec" || !strings.HasSuffix(joined, "\n--\nbe brief\n\nthe question") {
		t.Errorf("the arguments are %v", lines)
	}
	for _, want := range []string{"--ignore-user-config", "--ephemeral", "--model\ngpt-x", "--sandbox\nread-only", "--disable\nshell_tool"} {
		if !strings.Contains(joined, want) {
			t.Errorf("%q absent from %v", want, lines)
		}
	}
	for _, forbidden := range []string{"--dangerously-bypass-approvals-and-sandbox", "mcp_servers", "secret", "127.0.0.1"} {
		if strings.Contains(joined, forbidden) {
			t.Errorf("%q is among %v", forbidden, lines)
		}
	}
	if agenttest.IsDir(dir, root) {
		t.Errorf("started in %q, the folder the request names", dir)
	}
	if _, err := os.Stat(dir); err == nil {
		t.Errorf("the folder %q is still there", dir)
	}
}
