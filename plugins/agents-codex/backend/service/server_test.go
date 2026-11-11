package service

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun/agenttest"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/internal/adapter/codexcli"
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
	client := serveAgent(t, newAgentServer(false))

	programs, err := client.GetPrograms(t.Context(), []agentrun.ProgramCommand{{Program: "codex", Command: []string{appServer}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(programs) != 1 {
		t.Fatalf("the programs are %v", programs)
	}
	codex := programs[0]
	if codex.Name != "codex" || codex.Title != "OpenAI Codex" || !codex.IsInstalled || codex.IsDefault {
		t.Errorf("the programs are %v", programs)
	}
	if models := codex.Models; len(models) != 1 || models[0].Name != "gpt-x" ||
		models[0].Title != "GPT X" || !models[0].IsDefault {
		t.Errorf("Codex lists %v", codex.Models)
	}
}

// A program that is not installed is listed so, and Codex is not started for its models.
func TestAProgramThatIsNotInstalledListsNoModels(t *testing.T) {
	withoutAgentsAnywhere(t)
	client := serveAgent(t, newAgentServer(false))

	got, err := client.GetPrograms(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range got {
		if one.IsInstalled || len(one.Models) != 0 {
			t.Errorf("%v", one)
		}
	}
}

// A command line set by the installation counts as the program being installed, and its models are
// listed by running that command.
func TestGetAgentPrograms_ExplicitCommandInstallation(t *testing.T) {
	withoutAgentsAnywhere(t)
	appServer := agenttest.WriteScript(t, "codex",
		`echo '{"id":1,"result":{}}'`+"\n"+
			`echo '{"id":2,"result":{"data":[{"model":"gpt-x"}]}}'`+"\n"+
			"while read -r _; do :; done\n")
	client := serveAgent(t, newAgentServer(false))

	got, err := client.GetPrograms(t.Context(), []agentrun.ProgramCommand{{Program: "codex", Command: []string{appServer}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range got {
		if !one.IsInstalled || len(one.Models) != 1 {
			t.Errorf("codex is %v", one)
		}
	}
}

func withoutAgentsAnywhere(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	if found := codexcli.FindCodex(); found != "codex" {
		t.Skipf("Codex is installed at %s", found)
	}
}

// The run is started from the request: the command line, the folder, the model and the tools it is
// given.
func TestAskAgentStartsTheProgramFromTheRequest(t *testing.T) {
	root := t.TempDir()
	script, read := agenttest.RecordingProgram(t, "codex")

	agenttest.Ask(t, serveAgent(t, newAgentServer(false)), &agentrun.Request{
		Prompt:     "the question",
		Command:    []string{script, "--first"},
		Root:       root,
		Model:      "gpt-x",
		Tools:      []string{"note_read"},
		ToolsURL:   "http://127.0.0.1:1/mcp",
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
	stream, err := serveAgent(t, newAgentServer(false)).Ask(t.Context(), &agentrun.Request{
		Command: []string{script},
	})
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
func TestAskAgent_NoToolsWhenRequestToolsEmpty(t *testing.T) {
	root := t.TempDir()
	script, read := agenttest.RecordingProgram(t, "codex")

	agenttest.Ask(t, serveAgent(t, newAgentServer(false)), &agentrun.Request{
		Prompt:       "the question",
		Instructions: "be brief",
		Command:      []string{script},
		Root:         root,
		Model:        "gpt-x",
		ToolsURL:     "http://127.0.0.1:1/mcp",
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

func serveAgent(t *testing.T, agent *agentServer) *agentrun.Client {
	t.Helper()
	return agenttest.Serve(t, agent, agent)
}

func TestTheProgramOfTheConfiguredPathListsItsModels(t *testing.T) {
	agenttest.NeedsShell(t)
	t.Setenv("HOME", t.TempDir())
	script := agenttest.WriteScript(t, "codex",
		`echo '{"id":1,"result":{}}'`+"\n"+
			`echo '{"id":2,"result":{"data":[{"model":"m-custom","displayName":"Custom","isDefault":true}]}}'`+"\n"+
			"while read -r _; do :; done\n")
	s := newAgentServer(false)
	s.settings.Apply(map[string]string{"executablePath": " " + script + " "})
	s.settings.Apply(map[string]string{"unrelated": "val"})

	progs, err := s.GetPrograms(t.Context(), nil)
	if err != nil || len(progs) != 1 || !progs[0].IsInstalled || len(progs[0].Models) != 1 ||
		progs[0].Models[0].Name != "m-custom" || !progs[0].Models[0].IsDefault {
		t.Fatalf("GetPrograms with the configured path = (%+v, %v)", progs, err)
	}
}

func TestAsk_FirstSendFailure(t *testing.T) {
	sendErr := errors.New("send failed")

	err := newAgentServer(false).Ask(t.Context(), &agentrun.Request{Prompt: "q"}, func(*agentrun.Response) error {
		return sendErr
	})

	if !errors.Is(err, sendErr) {
		t.Errorf("Ask when the first send fails = %v, want %v", err, sendErr)
	}
}

func TestAskReportsAProgramThatCannotBeStarted(t *testing.T) {
	var steps []*agentrun.Response

	err := newAgentServer(false).Ask(t.Context(), &agentrun.Request{Prompt: "q", Command: []string{"/nonexistent-codex-bin"}}, func(r *agentrun.Response) error {
		steps = append(steps, r)
		return nil
	})

	if err != nil {
		t.Errorf("Ask when the start fails returned error: %v", err)
	}
	if len(steps) != 2 || steps[1].GetStopped() == "" {
		t.Errorf("Ask when the start fails steps = %+v, want ModelStarted and Stopped", steps)
	}
}
