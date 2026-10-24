package service

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun/agenttest"

	"connectrpc.com/connect"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// withoutAgentsAnywhere makes this machine hold none of the programs, wherever they are searched for.
func withoutAgentsAnywhere(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	was := antigravityPlaces
	antigravityPlaces = nil
	t.Cleanup(func() { antigravityPlaces = was })
}

// The program list names Antigravity, installed where it is found, with the models the program lists: none here.
func TestTheProgramListNamesAntigravityWithoutModels(t *testing.T) {
	agenttest.NeedsShell(t)
	t.Setenv("HOME", t.TempDir())
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "agy"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	client := agenttest.Serve(t, &agentServer{})

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
func TestGetAgentPrograms_ExplicitCommandInstallation(t *testing.T) {
	withoutAgentsAnywhere(t)
	client := agenttest.Serve(t, &agentServer{})

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

// The run is started from the request: the command line, the folder, the model and the prompt.
func TestAskAgentStartsTheProgramFromTheRequest(t *testing.T) {
	root := t.TempDir()
	script, read := agenttest.RecordingProgram(t, "agy")

	agenttest.Ask(t, agenttest.Serve(t, &agentServer{}), &numenv1.AskAgentRequest{
		Prompt:  "the question",
		Command: []string{script, "--first"},
		Root:    root,
		Model:   "gemini-x",
		Tools:   []string{"note_read"},
	})

	dir, lines := read()
	if lines[0] != "--first" || lines[1] != "-p" || lines[2] != "the question" {
		t.Errorf("the arguments are %v", lines)
	}
	for _, want := range []string{"--output-format", "stream-json", "--model", "gemini-x"} {
		if !slices.Contains(lines, want) {
			t.Errorf("%q absent from %v", want, lines)
		}
	}
	if !agenttest.IsDir(dir, root) {
		t.Errorf("started in %q, want %q", dir, root)
	}
}

// A home that cannot be set up is reported to the stream as the reason the run stopped.
func TestAskAgentReportsAHomeThatCannotBeSetUp(t *testing.T) {
	setUnusableTempDir(t)
	stream, err := agenttest.Serve(t, &agentServer{}).AskAgent(t.Context(), connect.NewRequest(&numenv1.AskAgentRequest{
		Command:  []string{"agy"},
		ToolsUrl: "http://127.0.0.1:1/mcp",
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
	if !strings.Contains(stopReason, "set up the Antigravity home") {
		t.Errorf("the run stopped with %q", stopReason)
	}
}

// A request naming no tool is one answer: the program is given no permission beforehand, no MCP
// server, an empty folder, and the instructions before the prompt.
func TestAskAgentOffersNoToolWhenRequestToolsEmpty(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(t.TempDir(), "run")
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".gemini", "config"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	userServers := `{"mcpServers":{"mine":{"serverUrl":"http://127.0.0.1:2/mcp"}}}`
	if err := os.WriteFile(filepath.Join(home, ".gemini", "config", "mcp_config.json"), []byte(userServers), 0o600); err != nil {
		t.Fatal(err)
	}
	script := agenttest.WriteScript(t, "agy",
		"pwd > "+agenttest.QuoteShell(out+".dir")+"\n"+
			"for a in \"$@\"; do printf '%s\\n' \"$a\"; done > "+agenttest.QuoteShell(out+".argv")+"\n"+
			"cat \"$HOME/.gemini/config/mcp_config.json\" > "+agenttest.QuoteShell(out+".mcp")+"\n")

	agenttest.Ask(t, agenttest.Serve(t, &agentServer{}), &numenv1.AskAgentRequest{
		Prompt:       "the question",
		Instructions: "be brief",
		Command:      []string{script},
		Root:         root,
		ToolsUrl:     "http://127.0.0.1:1/mcp",
		ToolsToken:   "secret",
	})

	argv, err := os.ReadFile(out + ".argv")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(argv), "-p\nbe brief\n\nthe question\n") {
		t.Errorf("the arguments are %q", argv)
	}
	if strings.Contains(string(argv), "--dangerously-skip-permissions") || strings.Contains(string(argv), "secret") {
		t.Errorf("the arguments are %q", argv)
	}
	mcp, err := os.ReadFile(out + ".mcp")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(mcp), "127.0.0.1") || strings.Contains(string(mcp), "numen") || strings.Contains(string(mcp), "mine") {
		t.Errorf("the program was given the servers %s", mcp)
	}
	dir, err := os.ReadFile(out + ".dir")
	if err != nil {
		t.Fatal(err)
	}
	if agenttest.IsDir(strings.TrimSpace(string(dir)), root) {
		t.Errorf("started in %q, the folder the request names", dir)
	}
}
