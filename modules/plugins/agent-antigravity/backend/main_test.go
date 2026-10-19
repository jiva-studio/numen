package main

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

// A program that is not installed is listed so.
func TestAProgramThatIsNotInstalledListsNoModels(t *testing.T) {
	withoutAgentsAnywhere(t)
	client := agenttest.Serve(t, &agentServer{})

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
func TestAProgramIsInstalledUnderTheCommandLineTheInstallationSet(t *testing.T) {
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
	t.Setenv("TMPDIR", "/nonexistent/tmp")
	stream, err := agenttest.Serve(t, &agentServer{}).AskAgent(t.Context(), connect.NewRequest(&numenv1.AskAgentRequest{
		Command:  []string{"agy"},
		ToolsUrl: "http://127.0.0.1:1/mcp",
	}))
	if err != nil {
		t.Fatal(err)
	}
	var stopped string
	for stream.Receive() {
		if reason := stream.Msg().GetStopped(); reason != "" {
			stopped = reason
		}
	}
	if !strings.Contains(stopped, "set up the Antigravity home") {
		t.Errorf("the run stopped with %q", stopped)
	}
}
