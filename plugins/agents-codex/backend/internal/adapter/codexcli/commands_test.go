package codexcli

import (
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/agents-codex/backend/usecase/task"
)

func TestAProgramOfNoneIsTheCodexFoundOnTheMachine(t *testing.T) {
	withoutAgentsAnywhere(t)

	if got := resolveProgram(""); got != "codex" {
		t.Errorf("resolveProgram = %q, want codex", got)
	}
}

func TestTheAnswerCommandTurnsOffTheFeaturesOfCodexThatReachOut(t *testing.T) {
	wantDisabled := []string{
		"shell_tool", "unified_exec", "view_image", "multi_agent", "apps", "browser_use",
		"computer_use", "plugins", "image_generation", "hooks",
	}
	if got := getDisabledFeatures(); !slices.Equal(got, wantDisabled) {
		t.Errorf("getDisabledFeatures = %v, want %v", got, wantDisabled)
	}
}

func TestTheModelIsOmittedWhenTheRequestAsksForTheDefault(t *testing.T) {
	if got := appendModelArguments(nil, task.Spec{Model: "default"}); len(got) != 0 {
		t.Errorf("appendModelArguments(default) = %v, want empty", got)
	}
}

func TestTheToolsCommandNamesNoMCPServerWithoutAToolsURL(t *testing.T) {
	cmd := Commands{}.MakeToolsCommand(task.Spec{Program: "/custom/bin/codex", Prompt: "hi"})

	if sliceContainsSubstr(cmd.Args, "mcp_servers") {
		t.Errorf("MakeToolsCommand without ToolsURL included mcp_servers: %v", cmd.Args)
	}
}

func TestTheCommandsRunTheProgramOfTheSpec(t *testing.T) {
	shPath, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not found")
	}

	tools := Commands{}.MakeToolsCommand(task.Spec{Program: shPath, Prompt: "p"})
	if tools.Path != shPath {
		t.Errorf("the tools command runs %q, want %q", tools.Path, shPath)
	}

	answer, err := Commands{}.MakeAnswerCommand(task.Spec{Program: shPath, Prompt: "p"})
	if err != nil || answer.Path != shPath || answer.Release == nil {
		t.Fatalf("the answer command is %q, hasCleanup=%v, err=%v", answer.Path, answer.Release != nil, err)
	}
	if err := answer.Release(); err != nil {
		t.Error(err)
	}
	if _, err := os.Stat(answer.Dir); err == nil {
		t.Errorf("the folder %q is still there after the release", answer.Dir)
	}
}

func sliceContainsSubstr(items []string, sub string) bool {
	for _, item := range items {
		if strings.Contains(item, sub) {
			return true
		}
	}
	return false
}

func TestTheAnswerCommandFailsWhenTheFolderCannotBeCreated(t *testing.T) {
	t.Setenv("TMPDIR", "/dev/null/nonexistent")
	if _, err := (Commands{}).MakeAnswerCommand(task.Spec{Prompt: "p"}); err == nil {
		t.Error("expected MakeAnswerCommand error when TMPDIR is invalid")
	}
}

// The prompt is read on the standard input, never from the command line, and a thread to continue is
// resumed after the options of the run; only a run that keeps no thread is ephemeral.
func TestThePromptIsTheInputAndAThreadIsResumed(t *testing.T) {
	prompt := "-- Plan the beds\n" + strings.Repeat("Raised beds of oak. ", 20)
	tools := Commands{}.MakeToolsCommand(task.Spec{Program: "/opt/codex", Prompt: prompt, Thread: "thread-beds", IsKept: true})
	answer, err := Commands{}.MakeAnswerCommand(task.Spec{Program: "/opt/codex", Prompt: prompt, IsKept: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := answer.Release(); err != nil {
			t.Error(err)
		}
	})
	single, err := Commands{}.MakeAnswerCommand(task.Spec{Program: "/opt/codex", Prompt: prompt})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := single.Release(); err != nil {
			t.Error(err)
		}
	})

	if got := tools.Args[len(tools.Args)-3:]; !slices.Equal(got, []string{"resume", "thread-beds", "-"}) || tools.Stdin != prompt {
		t.Errorf("the tools command ends in %q and reads %q", got, tools.Stdin)
	}
	if answer.Args[len(answer.Args)-1] != "-" || answer.Stdin != prompt || slices.Contains(answer.Args, "--ephemeral") {
		t.Errorf("the answer command is %q and reads %q", answer.Args, answer.Stdin)
	}
	if !slices.Contains(single.Args, "--ephemeral") {
		t.Errorf("a run that keeps no thread is %q", single.Args)
	}
	for _, args := range [][]string{tools.Args, answer.Args, single.Args} {
		if sliceContainsSubstr(args, "Plan the beds") {
			t.Errorf("the prompt is on the command line: %q", args)
		}
	}
}
