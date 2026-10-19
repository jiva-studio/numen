package main

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

	"connectrpc.com/connect"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// writeScript writes an executable shell script and answers with its path.
func writeScript(t *testing.T, name, body string) string {
	t.Helper()
	needsShell(t)
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// startAgent serves the agent over HTTP and answers with a client of it.
func startAgent(t *testing.T) numenv1connect.AgentServiceClient {
	t.Helper()
	srv := &agentServer{}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewAgentServiceHandler(srv))
	testServer := httptest.NewServer(mux)
	t.Cleanup(testServer.Close)
	return numenv1connect.NewAgentServiceClient(http.DefaultClient, testServer.URL)
}

// The program list names Codex, installed where it is found, with the models it lists under the command
// line the installation set for it.
func TestTheProgramListNamesCodexWithTheModelsItLists(t *testing.T) {
	needsShell(t)
	t.Setenv("HOME", t.TempDir())
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "codex"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	appServer := writeScript(t, "codex",
		`echo '{"id":1,"result":{}}'`+"\n"+
			`echo '{"id":2,"result":{"data":[{"model":"gpt-x","displayName":"GPT X","isDefault":true},{"model":"hid","hidden":true}]}}'`+"\n"+
			"while read -r _; do :; done\n")
	client := startAgent(t)

	got, err := client.GetAgentPrograms(t.Context(), connect.NewRequest(&numenv1.GetAgentProgramsRequest{
		Commands: []*numenv1.AgentProgramCommand{{Program: "codex", Command: []string{appServer}}},
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
	client := startAgent(t)

	got, err := client.GetAgentPrograms(t.Context(), connect.NewRequest(&numenv1.GetAgentProgramsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range got.Msg.GetPrograms() {
		if one.GetIsInstalled() || len(one.GetModels()) != 0 {
			t.Errorf("%v", one)
		}
	}
}

// What Codex listed is kept for as long as the command line stays the same.
func TestWhatCodexListedIsKeptWhileTheCommandStaysTheSame(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	counter := filepath.Join(t.TempDir(), "starts")
	command := writeScript(t, "codex",
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
	command := writeScript(t, "codex",
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

// The run is started from the request: the command line, the folder, the model and the tools it is
// given.
func TestAskAgentStartsTheProgramFromTheRequest(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(t.TempDir(), "run")
	script := writeScript(t, "codex",
		"pwd > "+out+".dir\n"+
			"for a in \"$@\"; do printf '%s\\n' \"$a\"; done > "+out+".argv\n")
	client := startAgent(t)

	stream, err := client.AskAgent(t.Context(), connect.NewRequest(&numenv1.AskAgentRequest{
		Prompt:     "the question",
		Command:    []string{script, "--first"},
		Root:       root,
		Model:      "gpt-x",
		ToolsUrl:   "http://127.0.0.1:1/mcp",
		ToolsToken: "secret",
	}))
	if err != nil {
		t.Fatal(err)
	}
	for stream.Receive() {
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}

	argv, err := os.ReadFile(out + ".argv")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(argv)), "\n")
	if lines[0] != "--first" || lines[1] != "exec" || lines[len(lines)-1] != "the question" {
		t.Errorf("the arguments are %v", lines)
	}
	for _, want := range []string{"--model", "gpt-x", `mcp_servers.numen.url="http://127.0.0.1:1/mcp"`} {
		if !slices.Contains(lines, want) {
			t.Errorf("%q absent from %v", want, lines)
		}
	}
	dir, err := os.ReadFile(out + ".dir")
	if err != nil {
		t.Fatal(err)
	}
	resolved, _ := filepath.EvalSymlinks(root)
	if got := strings.TrimSpace(string(dir)); got != root && got != resolved {
		t.Errorf("started in %q, want %q", got, root)
	}
}

// A command line the installation set counts as the program being installed, and its models are
// asked of that command.
func TestAProgramIsInstalledUnderTheCommandLineTheInstallationSet(t *testing.T) {
	withoutAgentsAnywhere(t)
	appServer := writeScript(t, "codex",
		`echo '{"id":1,"result":{}}'`+"\n"+
			`echo '{"id":2,"result":{"data":[{"model":"gpt-x"}]}}'`+"\n"+
			"while read -r _; do :; done\n")
	client := startAgent(t)

	got, err := client.GetAgentPrograms(t.Context(), connect.NewRequest(&numenv1.GetAgentProgramsRequest{
		Commands: []*numenv1.AgentProgramCommand{{Program: "codex", Command: []string{appServer}}},
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
// kept.
func TestAFailedCodexListingIsAskedForAgain(t *testing.T) {
	needsShell(t)
	t.Setenv("HOME", t.TempDir())
	failing := writeScript(t, "codex", "exit 1\n")
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

// needsShell skips a test that runs a script where the system has no shell to run it.
func needsShell(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fixtures are shell scripts")
	}
}

// withoutAgentsAnywhere makes this machine hold none of the programs, wherever they are searched for.
func withoutAgentsAnywhere(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	was := codexSearchPlaces
	codexSearchPlaces = nil
	t.Cleanup(func() { codexSearchPlaces = was })
}

// quoteShell wraps a string in single quotes with escaped internal single quotes for shell scripts.
func quoteShell(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
