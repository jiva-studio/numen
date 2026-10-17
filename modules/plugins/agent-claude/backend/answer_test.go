package main

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

const (
	scanInstruction   = "proofread scan"
	speechInstruction = "proofread speech"
	aQuestion         = "⟦0⟧the qulck brown fox ⟦1⟧jumped ovcr the lazy dog"
)

// recorder is a program standing in for the command line: it writes its arguments and its input
// into a folder, and answers with what it is told to. No test here runs the real program.
func recorder(t *testing.T, answer string) (command []string, wrote string) {
	t.Helper()
	wrote = t.TempDir()
	// An argument holds newlines, so the arguments are parted by a mark no argument carries.
	script := writeScript(t, "recorder",
		"for one in \"$@\"; do printf '%s\\036' \"$one\" >> "+wrote+"/argv; done\n"+
			"cat > "+wrote+"/stdin\n"+
			"ls -A . > "+wrote+"/there\n"+
			"printf '%s' '"+answer+"'\n")
	return []string{script}, wrote
}

func readFile(t *testing.T, dir, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// answerWith puts the question to the program and answers with the reply, else the error.
func answerWith(
	ctx context.Context, client numenv1connect.AgentServiceClient, request *numenv1.AnswerAgentRequest,
) (string, error) {
	reply, err := client.AnswerAgent(ctx, connect.NewRequest(request))
	if err != nil {
		return "", err
	}
	return reply.Msg.GetText(), nil
}

// The question goes on the input and the flags on the command line: the tool list after
// --disallowed-tools runs on for as many names as follow it, and a question is longer than a command
// line holds.
func TestAQuestionGoesOnTheInputAndTheFlagsOnTheCommandLine(t *testing.T) {
	command, wrote := recorder(t, "  0|the quick brown fox\n")
	client := startAgent(t, &agentServer{})

	text, err := answerWith(t.Context(), client, &numenv1.AnswerAgentRequest{
		Prompt: aQuestion, Instructions: scanInstruction, Model: "haiku", Command: command,
	})
	if err != nil {
		t.Fatal(err)
	}
	if text != "0|the quick brown fox" {
		t.Errorf("the answer came back as %q", text)
	}
	if got := readFile(t, wrote, "stdin"); got != aQuestion {
		t.Errorf("the input was %q", got)
	}

	argv := strings.Split(strings.TrimSuffix(readFile(t, wrote, "argv"), "\036"), "\036")
	want := append([]string{"-p", "--strict-mcp-config", "--model", "haiku", "--disallowed-tools"}, disallowed...)
	want = append(want, "--append-system-prompt", scanInstruction)
	if !slices.Equal(argv, want) {
		t.Errorf("started with\n%q\nand not\n%q", argv, want)
	}
}

// The program reads whatever standing instructions live where it was started, and those would land
// in the question.
func TestTheProgramIsStartedInAnEmptyFolder(t *testing.T) {
	command, wrote := recorder(t, "")
	client := startAgent(t, &agentServer{})

	if _, err := answerWith(t.Context(), client, &numenv1.AnswerAgentRequest{
		Prompt: aQuestion, Instructions: scanInstruction, Command: command,
	}); err != nil {
		t.Fatal(err)
	}
	if there := readFile(t, wrote, "there"); there != "" {
		t.Errorf("started among %q", there)
	}
}

// The caller says what is being corrected, and a model nobody named is not chosen.
func TestTheInstructionsAreTheCallersOwn(t *testing.T) {
	command, wrote := recorder(t, "")
	client := startAgent(t, &agentServer{})

	if _, err := answerWith(t.Context(), client, &numenv1.AnswerAgentRequest{
		Prompt: aQuestion, Instructions: speechInstruction, Command: command,
	}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(readFile(t, wrote, "argv"), speechInstruction) {
		t.Error("the instructions the caller gave were not appended")
	}
	if strings.Contains(readFile(t, wrote, "argv"), "--model") {
		t.Error("a model nobody named was chosen")
	}
}

// A question without instructions is refused, and so is one for a program nowhere to be found.
func TestAQuestionTheProgramCannotTakeIsRefused(t *testing.T) {
	client := startAgent(t, &agentServer{})
	if _, err := answerWith(t.Context(), client, &numenv1.AnswerAgentRequest{
		Prompt: aQuestion, Command: []string{"/bin/true"},
	}); err == nil {
		t.Error("a question without instructions was taken")
	}

	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("NUMEN_CLAUDE_COMMAND", "")
	t.Setenv("CLAUDE_COMMAND", "")
	if _, err := answerWith(t.Context(), client, &numenv1.AnswerAgentRequest{
		Prompt: aQuestion, Instructions: scanInstruction,
	}); err == nil {
		t.Error("a question was taken where the program is not installed")
	}
}

// A program that failed is an answer that failed, and says what the program said last.
func TestAProgramThatFailedIsToldByTheLastLineItWrote(t *testing.T) {
	script := writeScript(t, "refuses", "echo 'warming up' >&2\necho 'no model' >&2\nexit 1\n")
	client := startAgent(t, &agentServer{})

	text, err := answerWith(t.Context(), client, &numenv1.AnswerAgentRequest{
		Prompt: aQuestion, Instructions: scanInstruction, Command: []string{script},
	})
	if err == nil {
		t.Fatal("no error")
	}
	if text != "" {
		t.Errorf("got %q", text)
	}
	if !strings.Contains(err.Error(), "no model") || strings.Contains(err.Error(), "warming up") {
		t.Errorf("the error does not say what the program said last: %v", err)
	}
}

// The folder the program is started in is the run's own, and it goes when the run does however the
// run ended.
func TestTheFolderTheRunWasStartedInGoesWithIt(t *testing.T) {
	for _, one := range []struct {
		what string
		tail string
	}{
		{"a run that finished", "printf ''\n"},
		{"a run that failed", "exit 1\n"},
	} {
		t.Run(one.what, func(t *testing.T) {
			told := filepath.Join(t.TempDir(), "where")
			script := writeScript(t, "telling", "cat > /dev/null\npwd > "+told+"\n"+one.tail)
			client := startAgent(t, &agentServer{})

			_, _ = answerWith(t.Context(), client, &numenv1.AnswerAgentRequest{
				Prompt: aQuestion, Instructions: scanInstruction, Command: []string{script},
			})

			where := strings.TrimSpace(readFile(t, filepath.Dir(told), "where"))
			if where == "" {
				t.Fatal("the run said nothing about where it stood")
			}
			if _, err := os.Stat(where); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("%s is still there: %v", where, err)
			}
		})
	}
}

// A program the person stopped is a run that failed, and it ends when they stop it. Nothing it
// wrote before it was killed is an answer.
//
// The program starts programs of its own: one of them holding the output after the child is gone is
// what a wait would sit on.
func TestARunTheContextKilledEndsWithIt(t *testing.T) {
	script := writeScript(t, "dawdling", "cat > /dev/null\nprintf '0|the quick brown fox'\nsleep 60 &\nwait\n")
	client := startAgent(t, &agentServer{})

	ctx, stop := context.WithCancel(t.Context())
	go func() {
		time.Sleep(150 * time.Millisecond)
		stop()
	}()

	began := time.Now()
	text, err := answerWith(ctx, client, &numenv1.AnswerAgentRequest{
		Prompt: aQuestion, Instructions: scanInstruction, Command: []string{script},
	})
	took := time.Since(began)

	if err == nil {
		t.Fatal("a run that was killed came back with no reason")
	}
	if text != "" {
		t.Errorf("what a killed run wrote was taken as an answer: %q", text)
	}
	if took > 10*time.Second {
		t.Errorf("the run was stopped and went on for %v", took)
	}
}

// What a program that failed wrote on its output is not an answer: a refusal printed where a
// correction goes is not a correction.
func TestWhatAFailedRunWroteIsNotAnAnswer(t *testing.T) {
	script := writeScript(t, "refusing", "cat > /dev/null\nprintf 'I will not do that'\nexit 2\n")
	client := startAgent(t, &agentServer{})

	text, err := answerWith(t.Context(), client, &numenv1.AnswerAgentRequest{
		Prompt: aQuestion, Instructions: scanInstruction, Command: []string{script},
	})
	if err == nil {
		t.Fatal("a run that failed came back with no reason")
	}
	if text != "" {
		t.Errorf("what a failed run wrote was taken as an answer: %q", text)
	}
}

// A closed agent takes no question.
func TestAClosedAgentTakesNoQuestion(t *testing.T) {
	agentSvc := &agentServer{}
	_ = agentSvc.registry.Close()
	client := startAgent(t, agentSvc)

	if _, err := answerWith(t.Context(), client, &numenv1.AnswerAgentRequest{
		Prompt: aQuestion, Instructions: scanInstruction, Command: []string{"/bin/true"},
	}); err == nil {
		t.Error("a closed agent took a question")
	}
}
