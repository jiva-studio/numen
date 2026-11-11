package run

import (
	"context"
	"errors"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/rpcerror"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/model"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/ports"
)

type fixedCommands struct{ command string }

func (f fixedCommands) ResolveCommand(command []string) (string, []string) {
	if len(command) > 0 {
		return command[0], command[1:]
	}
	return f.command, nil
}

type echoReasoning struct{}

func (echoReasoning) GetReasoning(_, requested string) string { return requested }

type fixedPrompt string

func (f fixedPrompt) GetPrompt() string { return string(f) }

type fixedEnvironment struct{}

func (fixedEnvironment) GetVariables() []string { return []string{"PATH=/bin"} }

type fixedClock struct{}

func (fixedClock) Now() time.Time { return time.Unix(0, 0) }

// failingLauncher is the outer edge: a machine on which no program starts.
type failingLauncher struct{ launched []ports.Launch }

func (f *failingLauncher) Launch(_ context.Context, launch ports.Launch) (ports.Process, error) {
	f.launched = append(f.launched, launch)
	return nil, errors.New("start claude: no such file")
}

type recordingToolConfig struct{ writes int }

func (r *recordingToolConfig) Write(string, string) (string, error) {
	r.writes++
	return "/tmp/config.json", nil
}

func (r *recordingToolConfig) Remove(string) error { return nil }

func newRun(t *testing.T, command string, launcher ports.Launcher, toolConfig ports.ToolConfig, prompt string) *Run {
	t.Helper()
	r, err := New(Dependencies{
		Settings: new(settings.Settings), Prompt: fixedPrompt(prompt), Commands: fixedCommands{command: command},
		Reasoning: echoReasoning{}, Launcher: launcher, ToolConfig: toolConfig,
		Environment: fixedEnvironment{}, Clock: fixedClock{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestNewRefusesAMissingDependency(t *testing.T) {
	if _, err := New(Dependencies{}); err == nil {
		t.Error("a run without dependencies was created")
	}
}

func TestInstructionsAndSystemPromptAreJoinedOnALine(t *testing.T) {
	cases := []struct{ instructions, prompt, want string }{
		{"inst", "", "inst"},
		{"", "vault", "vault"},
		{"task", "vault", "task\nvault"},
		{"", "", ""},
	}
	for _, one := range cases {
		if got := combineInstructions(one.instructions, one.prompt); got != one.want {
			t.Errorf("combineInstructions(%q, %q) = %q, want %q", one.instructions, one.prompt, got, one.want)
		}
	}
}

// A task that names tools makes the file with the tools; the default model is left to the program.
func TestTheDefaultModelIsNotNamedToTheProgram(t *testing.T) {
	toolConfig := new(recordingToolConfig)
	r := newRun(t, "claude", new(failingLauncher), toolConfig, "")

	args, configFile, err := r.prepareArguments(&agentrun.Request{
		Model: model.DefaultName, Tools: []string{"note_read"}, ToolsURL: "http://127.0.0.1:1/mcp", ToolsToken: "tok",
	}, "", settings.Values{})
	if err != nil {
		t.Fatal(err)
	}
	if configFile != "/tmp/config.json" || toolConfig.writes != 1 {
		t.Errorf("config file %q written %d times", configFile, toolConfig.writes)
	}
	if slices.Contains(args, "--model") {
		t.Errorf("expected --model omitted for the default model, got %v", args)
	}
}

func TestATaskWithoutToolsWritesNoConfiguration(t *testing.T) {
	toolConfig := new(recordingToolConfig)
	r := newRun(t, "claude", new(failingLauncher), toolConfig, "")

	_, configFile, err := r.prepareArguments(&agentrun.Request{
		ToolsURL: "http://127.0.0.1:1/mcp", ToolsToken: "tok",
	}, "", settings.Values{})
	if err != nil || configFile != "" || toolConfig.writes != 0 {
		t.Fatalf("prepareArguments without tools = (%q, %v), %d writes", configFile, err, toolConfig.writes)
	}
}

func TestAskReportsAProgramThatIsNotInstalled(t *testing.T) {
	r := newRun(t, "", new(failingLauncher), new(recordingToolConfig), "")
	var steps []*agentrun.Response
	err := r.Ask(t.Context(), &agentrun.Request{Prompt: "q"}, func(step *agentrun.Response) error {
		steps = append(steps, step)
		return nil
	})
	if err != nil || len(steps) != 2 {
		t.Fatalf("Ask = %v, steps %v", err, steps)
	}
	if stopped, ok := steps[1].Step.(*agentrun.StoppedStep); !ok || stopped.Stopped != "claude not installed" {
		t.Errorf("the run ended with %v", steps[1])
	}
}

func TestAskReturnsTheErrorOfASendThatFails(t *testing.T) {
	sendErr := rpcerror.New(rpcerror.Canceled, os.ErrClosed)
	failing := func(*agentrun.Response) error { return sendErr }

	notInstalled := newRun(t, "", new(failingLauncher), new(recordingToolConfig), "")
	if err := notInstalled.Ask(t.Context(), &agentrun.Request{Prompt: "q"}, failing); err == nil {
		t.Error("expected error when send fails on the not-installed message")
	}

	installed := newRun(t, "claude", new(failingLauncher), new(recordingToolConfig), "")
	if err := installed.Ask(t.Context(), &agentrun.Request{Prompt: "q"}, failing); err == nil {
		t.Error("expected error when send fails on the first step")
	}
}

func TestAProgramThatDoesNotStartIsAnInternalError(t *testing.T) {
	launcher := new(failingLauncher)
	r := newRun(t, "claude", launcher, new(recordingToolConfig), "vault prompt")

	err := r.Ask(t.Context(), &agentrun.Request{Prompt: "q", Instructions: "task"}, func(*agentrun.Response) error { return nil })
	if rpcerror.GetCode(err) != rpcerror.Internal {
		t.Errorf("expected Internal error when the program does not start, got %v", err)
	}
	if len(launcher.launched) != 1 || launcher.launched[0].Command != "claude" || launcher.launched[0].Input != "q" {
		t.Errorf("launched %+v", launcher.launched)
	}
	if !slices.Contains(launcher.launched[0].Args, "task\nvault prompt") {
		t.Errorf("the instructions are not in %v", launcher.launched[0].Args)
	}
}

func TestAsk_ClosedRunIsUnavailable(t *testing.T) {
	r := newRun(t, "claude", new(failingLauncher), new(recordingToolConfig), "")
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	err := r.Ask(t.Context(), &agentrun.Request{Prompt: "q"}, func(*agentrun.Response) error { return nil })
	if rpcerror.GetCode(err) != rpcerror.Unavailable {
		t.Errorf("expected Unavailable from a closed run, got %v", err)
	}
}
