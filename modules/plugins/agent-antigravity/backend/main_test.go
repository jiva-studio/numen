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

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

func TestAgentServerToolsAndPrompt(t *testing.T) {
	srv := &agentServer{
		runner: &AgentRunner{Registry: &agentrun.Registry{}},
	}

	ctx := context.Background()

	// RegisterTool
	regResp, err := srv.RegisterTool(ctx, connect.NewRequest(&numenv1.RegisterToolRequest{
		Tool: &numenv1.AgentToolDefinition{
			Name:        "test_tool",
			Description: "A test tool",
		},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if regResp == nil || !regResp.Msg.GetSuccess() {
		t.Fatal("expected successful tool registration")
	}

	// ExtendSystemPrompt
	extResp, err := srv.ExtendSystemPrompt(ctx, connect.NewRequest(&numenv1.ExtendSystemPromptRequest{
		Extension: "Always be polite.",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if extResp.Msg.GetFullPrompt() != "Always be polite." {
		t.Errorf("expected prompt 'Always be polite.', got %q", extResp.Msg.GetFullPrompt())
	}

	// ExtendSystemPrompt a second time to append to existing prompt
	extResp2, err := srv.ExtendSystemPrompt(ctx, connect.NewRequest(&numenv1.ExtendSystemPromptRequest{
		Extension: "Be concise.",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if extResp2.Msg.GetFullPrompt() != "Always be polite.\nBe concise." {
		t.Errorf("expected prompt 'Always be polite.\\nBe concise.', got %q", extResp2.Msg.GetFullPrompt())
	}

	// GetAgentTools
	toolsResp, err := srv.GetAgentTools(ctx, connect.NewRequest(&numenv1.GetAgentToolsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(toolsResp.Msg.GetTools()) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(toolsResp.Msg.GetTools()))
	}
	if toolsResp.Msg.GetTools()[0].GetName() != "test_tool" {
		t.Errorf("expected tool test_tool, got %s", toolsResp.Msg.GetTools()[0].GetName())
	}
	if toolsResp.Msg.GetSystemPromptExtension() != "Always be polite.\nBe concise." {
		t.Errorf("expected prompt extension 'Always be polite.\\nBe concise.', got %q", toolsResp.Msg.GetSystemPromptExtension())
	}
}

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

// quoteShell wraps a string in single quotes with escaped internal single quotes for shell scripts.
func quoteShell(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// startAgent serves the agent over HTTP and answers with a client of it.
func startAgent(t *testing.T) numenv1connect.AgentServiceClient {
	t.Helper()
	srv := &agentServer{runner: &AgentRunner{Registry: &agentrun.Registry{}}}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewAgentServiceHandler(srv))
	testServer := httptest.NewServer(mux)
	t.Cleanup(testServer.Close)
	return numenv1connect.NewAgentServiceClient(http.DefaultClient, testServer.URL)
}

// A program that is not installed is listed so.
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

// The run is started from the request: the command line, the folder, the model and the tools it is
// given.
func TestAskAgentStartsTheProgramFromTheRequest(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(t.TempDir(), "run")
	script := writeScript(t, "agy",
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
	if lines[0] != "--first" || lines[1] != "-p" || lines[2] != "the question" {
		t.Errorf("the arguments are %v", lines)
	}
	for _, want := range []string{"--output-format", "stream-json", "--model", "gpt-x"} {
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
	was := antigravitySearchPlaces
	antigravitySearchPlaces = nil
	t.Cleanup(func() { antigravitySearchPlaces = was })
}

// The program list names Antigravity, installed where it is found, and lists no models.
func TestTheProgramListNamesAntigravityWithoutModels(t *testing.T) {
	needsShell(t)
	t.Setenv("HOME", t.TempDir())
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "agy"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	client := startAgent(t)

	got, err := client.GetAgentPrograms(t.Context(), connect.NewRequest(&numenv1.GetAgentProgramsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	programs := got.Msg.GetPrograms()
	if len(programs) != 1 {
		t.Fatalf("the programs are %v", programs)
	}
	agy := programs[0]
	if agy.GetName() != "antigravity" || agy.GetTitle() != "Antigravity" || !agy.GetIsInstalled() ||
		agy.GetIsDefault() || len(agy.GetModels()) != 0 {
		t.Errorf("the programs are %v", programs)
	}
}

// A command line the installation set counts as the program being installed.
func TestAProgramIsInstalledUnderTheCommandLineTheInstallationSet(t *testing.T) {
	withoutAgentsAnywhere(t)
	client := startAgent(t)

	got, err := client.GetAgentPrograms(t.Context(), connect.NewRequest(&numenv1.GetAgentProgramsRequest{
		Commands: []*numenv1.AgentProgramCommand{{Program: "antigravity", Command: []string{"/opt/agy"}}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if programs := got.Msg.GetPrograms(); len(programs) != 1 || !programs[0].GetIsInstalled() {
		t.Errorf("the programs are %v", programs)
	}
}
