package claudecode

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

	"github.com/jiva-studio/numen/modules/libs/core/proofread"
)

// recorder is a program standing in for the command line: it writes its
// arguments and its input into a folder, and answers with what it is told to.
//
// No test here runs the real command line.
func recorder(t *testing.T, answer string) (command []string, wrote string) {
	t.Helper()
	wrote = t.TempDir()
	script := filepath.Join(t.TempDir(), "recorder")
	// An argument holds newlines, so the arguments are parted by a mark no
	// argument carries.
	written := "#!/bin/sh\n" +
		"for one in \"$@\"; do printf '%s\\036' \"$one\" >> " + wrote + "/argv; done\n" +
		"cat > " + wrote + "/stdin\n" +
		"ls -A . > " + wrote + "/there\n" +
		"printf '%s' '" + answer + "'\n"
	if err := os.WriteFile(script, []byte(written), 0o755); err != nil {
		t.Fatal(err)
	}
	return []string{script}, wrote
}

func held(t *testing.T, dir, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

var aBatch = proofread.Batch{Number: 7, Lines: []proofread.Line{
	{Number: 0, Text: "the qulck brown fox"},
	{Number: 1, Text: "jumped ovcr the lazy dog"},
}}

// The batch goes on the input and the flags on the command line: the tool list
// after --disallowed-tools runs on for as many names as follow it, and a batch
// is longer than a command line holds.
func TestABatchGoesOnTheInputAndTheFlagsOnTheCommandLine(t *testing.T) {
	command, wrote := recorder(t, "0|the quick brown fox")
	by := &Proofreader{Command: command, Model: "haiku", Instruction: proofread.ScanInstruction}

	out, err := by.Proofread(t.Context(), []proofread.Batch{aBatch})
	if err != nil {
		t.Fatal(err)
	}
	if out[7] != "0|the quick brown fox" {
		t.Errorf("the reply came back as %q", out[7])
	}

	if got := held(t, wrote, "stdin"); got != proofread.Ask(aBatch) {
		t.Errorf("the input was %q", got)
	}

	argv := strings.Split(strings.TrimSuffix(held(t, wrote, "argv"), "\036"), "\036")
	want := append([]string{"-p", "--strict-mcp-config", "--model", "haiku", "--disallowed-tools"},
		disallowed...)
	want = append(want, "--append-system-prompt", proofread.ScanInstruction)
	if !slices.Equal(argv, want) {
		t.Errorf("started with\n%q\nand not\n%q", argv, want)
	}
}

// The command line reads whatever standing instructions live where it was
// started, and those would land in the proofreading prompt.
func TestTheCommandLineIsStartedInAnEmptyFolder(t *testing.T) {
	command, wrote := recorder(t, "")
	by := &Proofreader{Command: command, Model: "haiku", Instruction: proofread.ScanInstruction}

	if _, err := by.Proofread(t.Context(), []proofread.Batch{aBatch}); err != nil {
		t.Fatal(err)
	}

	if there := held(t, wrote, "there"); there != "" {
		t.Errorf("started among %q", there)
	}
}

// The caller says what is being corrected: a scan and speech are corrected for
// different mistakes.
func TestTheInstructionIsTheCallersOwn(t *testing.T) {
	command, wrote := recorder(t, "")
	by := &Proofreader{Command: command, Instruction: proofread.SpeechInstruction}

	if _, err := by.Proofread(t.Context(), []proofread.Batch{aBatch}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(held(t, wrote, "argv"), proofread.SpeechInstruction) {
		t.Error("the instruction the caller gave was not appended")
	}
	if strings.Contains(held(t, wrote, "argv"), "--model") {
		t.Error("a model nobody named was chosen")
	}
}

// What put a correction right stands beside every line it made, and a model is
// what that is.
func TestTheNameSaysWhichModelCorrected(t *testing.T) {
	if got := (&Proofreader{Model: "haiku"}).Name(); !strings.Contains(got, "haiku") {
		t.Errorf("got %q", got)
	}
}

// A command line that failed is a run that failed: the batches already answered
// are dropped and the caller asks again.
func TestACommandLineThatFailedEndsTheRun(t *testing.T) {
	script := filepath.Join(t.TempDir(), "refuses")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho 'no model' >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	by := &Proofreader{Command: []string{script}, Instruction: proofread.ScanInstruction}

	out, err := by.Proofread(t.Context(), []proofread.Batch{aBatch})
	if err == nil {
		t.Fatal("no error")
	}
	if out != nil {
		t.Errorf("got %v", out)
	}
	if !strings.Contains(err.Error(), "no model") {
		t.Errorf("the error does not say what the command line said: %v", err)
	}
}

// The folder the command line is started in is the run's own, and it goes when
// the run does however the run ended.
func TestTheFolderTheRunWasStartedInGoesWithIt(t *testing.T) {
	for _, one := range []struct {
		what   string
		script string
	}{
		{"a run that finished", "#!/bin/sh\ncat > /dev/null\npwd > " + "%s" + "\nprintf ''\n"},
		{"a run that failed", "#!/bin/sh\ncat > /dev/null\npwd > " + "%s" + "\nexit 1\n"},
	} {
		t.Run(one.what, func(t *testing.T) {
			told := filepath.Join(t.TempDir(), "where")
			script := filepath.Join(t.TempDir(), "telling")
			written := strings.Replace(one.script, "%s", told, 1)
			if err := os.WriteFile(script, []byte(written), 0o755); err != nil {
				t.Fatal(err)
			}
			by := &Proofreader{Command: []string{script}, Instruction: proofread.ScanInstruction}

			_, _ = by.Proofread(t.Context(), []proofread.Batch{aBatch})

			where := strings.TrimSpace(held(t, filepath.Dir(told), "where"))
			if where == "" {
				t.Fatal("the run said nothing about where it stood")
			}
			if _, err := os.Stat(where); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("%s is still there: %v", where, err)
			}
		})
	}
}

// A command line the person stopped is a run that failed, and it ends when they
// stop it. Nothing it wrote before it was killed is an answer.
//
// The command line starts programs of its own: one of them holding the output
// after the child is gone is what a wait would sit on.
func TestARunTheContextKilledEndsWithIt(t *testing.T) {
	script := filepath.Join(t.TempDir(), "dawdling")
	written := "#!/bin/sh\ncat > /dev/null\nprintf '0|the quick brown fox'\nsleep 60 &\nwait\n"
	if err := os.WriteFile(script, []byte(written), 0o755); err != nil {
		t.Fatal(err)
	}
	by := &Proofreader{Command: []string{script}, Instruction: proofread.ScanInstruction}

	ctx, stop := context.WithCancel(context.Background())
	go func() {
		time.Sleep(150 * time.Millisecond)
		stop()
	}()

	began := time.Now()
	out, err := by.Proofread(ctx, []proofread.Batch{aBatch})
	took := time.Since(began)

	if err == nil {
		t.Fatal("a run that was killed came back with no reason")
	}
	if out != nil {
		t.Errorf("what a killed run wrote was taken as an answer: %v", out)
	}
	if took > 10*time.Second {
		t.Errorf("the run was stopped and went on for %v", took)
	}
}

// What a command line that failed wrote on its output is not an answer: a
// refusal printed where a correction goes is not a correction.
func TestWhatAFailedRunWroteIsNotAnAnswer(t *testing.T) {
	script := filepath.Join(t.TempDir(), "refusing")
	written := "#!/bin/sh\ncat > /dev/null\nprintf 'I will not do that'\nexit 2\n"
	if err := os.WriteFile(script, []byte(written), 0o755); err != nil {
		t.Fatal(err)
	}
	by := &Proofreader{Command: []string{script}, Instruction: proofread.ScanInstruction}

	out, err := by.Proofread(t.Context(), []proofread.Batch{aBatch})
	if err == nil {
		t.Fatal("a run that failed came back with no reason")
	}
	if out != nil {
		t.Errorf("what a failed run wrote was taken as an answer: %v", out)
	}
}
